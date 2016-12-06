package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	//	"io/ioutil"
	"net/http"
	//	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/toming90/containerpilot/commands"
	"github.com/toming90/containerpilot/discovery"
	"github.com/toming90/containerpilot/utils"

	consulAPI "github.com/hashicorp/consul/api"
)

const (
	SLASH = "/"
)

type Storage struct {
	Path             string      `mapstructure:"path"`
	Poll             int         `mapstructure:"poll"` // time in seconds
	OnChangeExec     interface{} `mapstructure:"onChange"`
	Timeout          string      `mapstructure:"timeout"`
	OnChangePostUrl  string      `mapstructure:"onChangePostUrl"`
	discoveryService discovery.ServiceBackend
	onChangeCmd      *commands.Command
}

// KeyValueChangeUnit stores changes of paths
type KeyValueChangeUnit struct {
	Path  string `json:"path_on_change"`
	Value string `json:"value"`
}

type KeyValueChanges struct {
	Root         string                `json:"root"`
	Addition     *[]KeyValueChangeUnit `json:"addition,omitempty"`
	Modification *[]KeyValueChangeUnit `json:"modification,omitempty"`
	Deletion     *[]KeyValueChangeUnit `json:"deletion,omitempty"`
}

// store a map of last key-value changes
var keyValueMaps = make(map[string](map[string]string))

// NewStorage create new storage config
func NewStorages(raw []interface{}, disc discovery.ServiceBackend) ([]*Storage, error) {
	if raw == nil {
		return []*Storage{}, nil
	}
	var storages []*Storage
	if err := utils.DecodeRaw(raw, &storages); err != nil {
		return nil, fmt.Errorf("storage configuration error: %v", err)
	}
	for _, s := range storages {
		/* check whether path is empty */
		if s.Path == "" {
			return nil, fmt.Errorf("NewStorage[storage/keystore.go] storage must have a `path`")
		}
		/* add workspace prefix for path */
		if strings.HasPrefix(s.Path, SLASH) {
			s.Path = s.Path[1:]
		}
		/* parse onChange */
		//		cmd, err := utils.ParseCommandArgs(s.OnChangeExec)
		//		if err != nil {
		//			return nil, fmt.Errorf("NewStorage[storage/keystore.go] could not parse `onChange` in storage %s: %s",
		//				s.Path, err)
		//		}
		var cmd *commands.Command
		if s.OnChangeExec != nil {
			c, err := commands.NewCommand(s.OnChangeExec, s.Timeout)
			if err != nil {
				return nil, fmt.Errorf("Could not parse `onChange` in backend %s: %s",
					s.Path, err)
			}
			cmd = c
		}

		/* if interval poll is too short, we stop it */
		if s.Poll < 1 {
			return nil, fmt.Errorf("NewStorages[storage/keystore.go] `poll` must be > 0 in storage %s",
				s.Path)
		}
		s.onChangeCmd = cmd
		s.discoveryService = disc
	}
	return storages, nil
}

// PollStop does nothing in a Storage
func (s *Storage) PollStop() {
	// Nothing to do
}

// PollTime implements Pollable for Storage
// It returns the backend's poll interval.
func (s Storage) PollTime() time.Duration {
	return time.Duration(s.Poll) * time.Second
}

// PollAction implements Pollable for Storage.
// If the values in the path have changed since the last run,
// we fire the on change handler.
func (s *Storage) PollAction() {
	log.Debugln("PollAction[storage/keystore.go] start.")
	/* check if values in storage path has changed */
	isChanged, changes := s.checkForKeyValueChanges(keyValueMaps[s.Path])
	if isChanged {
		if s.onChangeCmd != nil {
			log.Infof("PollAction[storage/keystore.go] Triggering onChange cmd: %v", s.onChangeCmd.Args)
			/* run on-change cmd */
			s.onChange()
		}
		if s.OnChangePostUrl != "" {
			log.Infof("PollAction[storage/keystore.go] Posting changes to specfied url: %v", s.OnChangePostUrl)
			/* post changes to specified url*/
			s.postKeyValuePairsOnChange(changes)
		}
	}
}

// checkForKeyValueChanges compares key-value pairs from Consul to those from last time
// if there is changes, it will trigger function PostKeyValuePairsOnChange and
// return true
func (s *Storage) checkForKeyValueChanges(old_kv_map map[string]string) (bool, *KeyValueChanges) {
	/* get consul client */
	var c consulAPI.Client
	if consulCli, ok := s.discoveryService.GetClient().(consulAPI.Client); !ok {
		log.Error("[keystore.go]: Cannot get Consul Client")
	} else {
		c = consulCli
	}
	/* get consul ket values */
	kv := c.KV()

	log.Debugf("checkForKeyValueChanges[storage/keystore.go] Checking changes under path: %s\n", s.Path)

	/* get all key values under the path by consul api */
	if lists, _, e := kv.List(s.Path, nil); e != nil {
		log.Errorf("checkForKeyValueChanges[storage/keystore.go] Cannot get key store updates from Consul: %v\n", e.Error())
		return false, nil
	} else {
		/* create 3 list representing additions, deletions, modifications of key-store changes */
		adds := make([]KeyValueChangeUnit, 0)
		dels := make([]KeyValueChangeUnit, 0)
		mods := make([]KeyValueChangeUnit, 0)

		/* create KeyValueChanges struct */
		chgs := &KeyValueChanges{Root: s.Path, Addition: &adds, Modification: &mods, Deletion: &dels}

		/* create new key-value map */
		var new_kv_map = make(map[string]string)

		/* loop key value pairs that fetch from consul
		and generate new key-value map */
		for _, p := range lists {
			valnew := string(p.Value[:])
			_, oke := old_kv_map[p.Key]

			// addition
			if !oke {
				new_kv_map[p.Key] = valnew
				adds = append(adds, KeyValueChangeUnit{Path: p.Key, Value: valnew})

				// either modification or unchanged
			} else {
				// modification
				if valnew != old_kv_map[p.Key] {
					mods = append(mods, KeyValueChangeUnit{Path: p.Key, Value: valnew})
				}
				new_kv_map[p.Key] = valnew

				// remove keys that are modified or unchanged in old_kv_map
				delete(old_kv_map, p.Key)
			}
		}

		for k, _ := range old_kv_map {
			dels = append(dels, KeyValueChangeUnit{Path: k, Value: ""})
		}

		keyValueMaps[s.Path] = new_kv_map
		if len(adds) == 0 && len(dels) == 0 && len(mods) == 0 {
			log.Debugf("checkForKeyValueChanges[storage/keystore.go] No changes under path: %s\n", s.Path)
			return false, nil
		} else {
			log.Infof("checkForKeyValueChanges[storage/keystore.go] Detect changes under path: %s\n", s.Path)
			return true, chgs
		}
	}
}

//postKeyValuePairsOnChange posts changes of keu-value pair in form of json to certain url
func (storage *Storage) postKeyValuePairsOnChange(chgs *KeyValueChanges) {
	url := strings.Trim(storage.OnChangePostUrl, " ")
	/* if url is empty, do not post anything */
	if url == "" {
		log.Errorln("postKeyValuePairsOnChange[storage/keystore.go] Error: url is empty.")
		return
	}
	/* convert changes to json format */
	jsonStr, e := json.Marshal(chgs)
	if e != nil {
		log.Errorf("postKeyValuePairsOnChange[storage/keystore.go] Found error whilie building json: %v", e)
		return
	}

	log.Debugf("postKeyValuePairsOnChange[storage/keystore.go] Changes in json format:%v\n", string(jsonStr))
	req, e := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if e != nil {
		log.Errorf("postKeyValuePairsOnChange[storage/keystore.go] Found eroor while making post request: %v\n", e)
		return
	}

	/* set response data type to json */
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	if resp, e := client.Do(req); e != nil {
		log.Errorf("postKeyValuePairsOnChange[storage/keystore.go] Found eroor: %v\n", e)
	} else {
		defer resp.Body.Close()
		log.Infof("postKeyValuePairsOnChange[storage/keystore.go] Response Status:%v\n", resp.Status)
	}
}

// OnChange runs the storage's onChange command, returning the results
func (s *Storage) onChange() error {

	return commands.RunWithTimeout(s.onChangeCmd, log.Fields{
		"process": "onChange", "Path": s.Path})

}

// CreateDefaultFoldersAndLinks create two folders 'log' and 'data' under /log
// And link '/log' and '/data' to them respectively.
func CreateDefaultFoldersAndLinks() {
	/*
		mkdir -p /tmp/log
		mkdir -p /tmp/data
		ln -snf /tmp/log /log
		ln -snf /tmp/data /data
	*/

	c1 := exec.Command("mkdir", "-p", "/tmp/log")
	if e := c1.Start(); e != nil {
		log.Fatal(e)
	}
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Waiting for command to finish: %v %v\n", c1.Path, c1.Args)
	if e := c1.Wait(); e != nil {
		log.Printf("Command finished with error: %v", e)
	}
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Successfully executed cmd: %v %v\n", c1.Path, c1.Args)

	c2 := exec.Command("mkdir", "-p", "/tmp/data")
	if e := c2.Start(); e != nil {
		log.Fatal(e)
	}
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Waiting for command to finish: %v %v\n", c2.Path, c2.Args)
	if e := c2.Wait(); e != nil {
		log.Printf("Command finished with error: %v", e)
	}
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Successfully executed cmd: %v %v\n", c2.Path, c2.Args)

	c3 := exec.Command("ln", "-snf", "/tmp/log", "/log")
	if e := c3.Start(); e != nil {
		log.Fatal(e)
	}
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Waiting for command to finish: %v %v\n", c3.Path, c3.Args)
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Successfully executed cmd: %v %v\n", c3.Path, c3.Args)
	if e := c3.Wait(); e != nil {
		log.Printf("Command finished with error: %v", e)
	}

	c4 := exec.Command("ln", "-snf", "/tmp/data", "/data")
	if e := c4.Start(); e != nil {
		log.Fatal(e)
	}
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Waiting for command to finish: %v %v\n", c4.Path, c4.Args)
	log.Infof("CreateDefaultFoldersAndLinks[storage/keystore.go] Successfully executed cmd: %v %v\n", c4.Path, c4.Args)
	if e := c4.Wait(); e != nil {
		log.Printf("Command finished with error: %v", e)
	}
}
