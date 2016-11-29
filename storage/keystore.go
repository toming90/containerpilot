package custom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/toming90/containerpilot/discovery/consul"
	"github.com/toming90/containerpilot/utils"
)

const (
	COBALT_WS = "COBALT_WS"
	SLASH     = "/"
)

var workspacePrefix string

func init() {
	workspacePrefix = os.Getenv(COBALT_WS)
	if workspacePrefix == "" {
		log.Errorln("Cannot Detect Environment Variable: COBALT_WS")
		os.Exit(1)
	}
}

type Storage struct {
	Path            string      `mapstructure:"path"`
	Poll            int         `mapstructure:"poll"` // time in seconds
	OnChangeExec    interface{} `mapstructure:"onChange"`
	onChangeCmd     *exec.Cmd
	OnChangePostUrl string `mapstructure:"onChangePostUrl"`
	consul          consul.Consul
}

func NewStorage(raw json.RawMessage, consulCli discovery.Consul) ([]*Storage, error) {
	if raw == nil {
		return []*Storage{}, nil
	}
	storage := make([]*Storage, 0)
	if err := json.Unmarshal(raw, &storage); err != nil {
		return nil, fmt.Errorf("storage configuration error: %v", err)
	}
	for _, s := range storage {
		if s.Path == "" {
			return nil, fmt.Errorf("storage must have a `path`")
		}
		if strings.HasPrefix(s.Path, SLASH) {
			s.Path = workspacePrefix + s.Path
		} else {
			s.Path = workspacePrefix + SLASH + s.Path
		}

		cmd, err := utils.ParseCommandArgs(s.OnChangeExec)
		if err != nil {
			return nil, fmt.Errorf("Could not parse `onChange` in storage %s: %s",
				s.Path, err)
		}

		if s.Poll < 1 {
			return nil, fmt.Errorf("`poll` must be > 0 in storage %s",
				s.Path)
		}
		s.onChangeCmd = cmd
		s.consul = consulCli
	}
	return storage, nil
}

// PollTime implements Pollable for Storage
// It returns the backend's poll interval.
func (s Storage) PollTime() time.Duration {
	return time.Duration(s.Poll) * time.Second
}

// PollAction implements Pollable for Storage.
// If the values in the discovery service have changed since the last run,
// we fire the on change handler.
func (s *Storage) PollAction() {
	log.Info("PollAction[custom/keystore.go] PollAction: start.")
	//	if .CheckForUpstreamChanges() {
	//		b.OnChange()
	//	}
	//	storage := pollable.(*StorageConfig) // if we pass a bad type here we crash intentionally
	//	if storage.CheckForKeyValueChanges() {
	//		run(storage.onChangeCmd)
	//	}
	//	s.discoveryService.
	if s.CheckForKeyValueChanges(keyValueMaps[s.Path]) && s.onChangeCmd != nil {
		s.OnChange()
	}

}

//func (s *StorageConfig) CheckForKeyValueChanges() bool {
//	return s.discoveryService.CheckForKeyValueChanges(keyValueMaps[s.Path], s)
//}

// PollStop does nothing in a Storage
func (b *Storage) PollStop() {
	// Nothing to do
}

//ALL THE BELOW ARE CUSTOMIZATION

//type ExtraConfig struct {
//	Consul   string           `json:"consul,omitempty"`
//	Storages []*StorageConfig `json:"storage"`
//}

//type StorageConfig struct {
//	Path         string          `json:"path"`
//	Poll         int             `json:"poll"` // time in seconds
//	OnChangeExec json.RawMessage `json:"onChange,omitempty"`
//	onChangeCmd  *exec.Cmd

//	OnChangePostUrl string `json:"onChangePostUrl,omitempty"`

//	discoveryService ExtraDiscoveryService
//}

//type ExtraDiscoveryService interface {

//	//to detect kv pair changes
//	CheckForKeyValueChanges(map[string]string, *StorageConfig) bool

//	//initial sync kv pairs
//	InitSyncKeyValues(*StorageConfig)
//}

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

//// store a map of last key-value changes
var keyValueMaps = make(map[string](map[string]string))

//// load storage config file from the path set by -config flag
//func loadExtraConfig() *ExtraConfig {

//	var discovery ExtraDiscoveryService
//	discoveryCount := 0

//	extraConfigFlag := flag.Lookup("config").Value.String()

//	if extraConfigFlag == "" {
//		extraConfigFlag = os.Getenv("CONTAINERBUDDY")
//	}

//	config, err := parseExtraConfig(extraConfigFlag)
//	if err != nil {
//		log.Fatal(err)
//	}

//	for _, discoveryBackend := range []string{"Consul"} {
//		switch discoveryBackend {
//		case "Consul":
//			if config.Consul != "" {
//				config.Consul = os.ExpandEnv(config.Consul)
//				fmt.Printf("Consul value is:  %s\n", config.Consul)
//				discovery = NewConsulConfig(config.Consul)
//				discoveryCount += 1
//			}
//		}
//	}

//	if discoveryCount == 0 {
//		log.Fatal("No discovery backend defined")
//	} else if discoveryCount > 1 {
//		log.Fatal("More than one discovery backend defined")
//	}

//	for _, storage := range config.Storages {
//		storage.Path = os.ExpandEnv(storage.Path)

//		//storage.onChangeArgs = strings.Split(storage.OnChangeExec, " ")
//		cmd, _ := parseCommandArgs(storage.OnChangeExec)

//		fmt.Printf("StoragePath=%s\n", storage.Path)

//		fmt.Printf("onChangeExec=%s\n", cmd)

//		storage.discoveryService = discovery

//		storage.onChangeCmd = cmd
//	}

//	return config

//}

//// parse a config file to Go Struct
//func parseExtraConfig(configFlag string) (*ExtraConfig, error) {
//	if configFlag == "" {
//		log.Fatal("-config flag is required.")
//	}

//	var data []byte
//	if strings.HasPrefix(configFlag, "file://") {
//		var err error
//		if data, err = ioutil.ReadFile(strings.SplitAfter(configFlag, "file://")[1]); err != nil {
//			log.Fatalf("Could not read config file: %s", err)
//		}
//	} else {
//		data = []byte(configFlag)
//	}

//	//	config := &ExtraConfig{}

//	//	if err := json.Unmarshal(data, &config); err != nil {
//	//		log.Fatalf("Could not parse configuration: %s", err)
//	//	}

//	//	return config
//	template, err := ApplyTemplate(data)
//	if err != nil {
//		return nil, fmt.Errorf(
//			"Could not apply template to config: %s", err)
//	}
//	return unmarshalExtraConfig(template)
//}

//// Implements `pollingFunc`; args are the executable we use to check the
//// application KV change. If the error code on that exectable is
//// 0, we write a TTL health check to the health check store.
//func checkForKvChange(pollable Pollable) {
//	storage := pollable.(*StorageConfig) // if we pass a bad type here we crash intentionally
//	if storage.CheckForKeyValueChanges() {
//		run(storage.onChangeCmd)
//	}
//}

//func (s StorageConfig) PollTime() int {
//	return s.Poll
//}

//func (s *StorageConfig) CheckForKeyValueChanges() bool {
//	return s.discoveryService.CheckForKeyValueChanges(keyValueMaps[s.Path], s)
//}

//func (s *StorageConfig) initialKeyValueUpdate() {
//	s.discoveryService.InitSyncKeyValues(s)
//}

//// first time use only, sync key-value pairs from Consul to local
//func (c Consul) InitSyncKeyValues(storage *StorageConfig) {
//	kv := c.KV()
//	if lists, _, err := kv.List(storage.Path, nil); err != nil {
//		log.Println("Found Error - init sync keystore:")
//		log.Println(err)

//	} else {
//		log.Printf("Initial Sync KeyValues Under Path %s...\n", storage.Path)

//		var new_kv_map = make(map[string]string)
//		for _, p := range lists {
//			new_kv_map[p.Key] = string(p.Value[:])
//		}
//		keyValueMaps[storage.Path] = new_kv_map
//	}

//}

//// compare key-value pairs from Consul to those from last time
//// if there is changes, it will trigger function PostKeyValuePairsOnChange and
//// return true
func (storage *Storage) CheckForKeyValueChanges(old_kv_map map[string]string) bool {

	c := storage.consul

	kv := c.KV()

	log.Warnf("Checking Key-Value Pair Changes Under Path \n%s\n", storage.Path)

	if lists, _, err := kv.List(storage.Path, nil); err != nil {
		log.Println("Found Error - check keystore update:")
		log.Println(err)
		return false

	} else {

		adds := make([]KeyValueChangeUnit, 0)
		dels := make([]KeyValueChangeUnit, 0)
		mods := make([]KeyValueChangeUnit, 0)

		chgs := &KeyValueChanges{Root: storage.Path, Addition: &adds, Modification: &mods, Deletion: &dels}

		var new_kv_map = make(map[string]string)

		// loop new pairs
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

		keyValueMaps[storage.Path] = new_kv_map
		if len(adds) == 0 && len(dels) == 0 && len(mods) == 0 {
			log.Warnf("No key-value pair(s) Changed Under Path %s...\n", storage.Path)
			return false
		} else {
			log.Warnf("Detect key-value pair(s) Changed Under Path %s...\n", storage.Path)
			PostKeyValuePairsOnChange(storage.OnChangePostUrl, chgs)
			return true
		}
	}
}

////// post changes of keu-value pair in form of json to certain url
func PostKeyValuePairsOnChange(url string, chgs *KeyValueChanges) {
	if strings.Trim(url, " ") != "" {

		log.Warnf("Posting changes in key-value pair(s) to url: %s...\n", url)

		if jsonStr, err := json.Marshal(chgs); err != nil {
			log.Println("Found Error While Building JSON:")
			fmt.Println(err)
		} else {

			//bytes.NewBuffer(jsonStr)
			log.Printf("Posting Json:%v\n", string(jsonStr[:]))
			if req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr)); err != nil {
				log.Println("Found Error Making Post Request:")
				log.Println(err)
			} else {

				//req.Header.Set("X-Custom-Header", "myvalue")
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{}
				if resp, err := client.Do(req); err != nil {
					log.Println("Found Error:")
					log.Println(err)
				} else {
					defer resp.Body.Close()
					log.Printf("response Status:%v\n", resp.Status)
					log.Printf("response Headers:%v\n", resp.Header)
					body, _ := ioutil.ReadAll(resp.Body)
					log.Printf("response Body:%v\n", string(body))
				}

			}
		}
	} else {
		log.Printf("No Url to Post.\n")
	}
}

// OnChange runs the backend's onChange command, returning the results
func (s *Storage) OnChange() (int, error) {
	defer func() {
		// reset command object because it can't be reused
		s.onChangeCmd = utils.ArgsToCmd(s.onChangeCmd.Args)
	}()

	exitCode, err := utils.RunWithFields(s.onChangeCmd, log.Fields{"process": "OnChange", "storage": s.Path})
	return exitCode, err
}

//func unmarshalExtraConfig(data []byte) (*ExtraConfig, error) {
//	extraConfig := &ExtraConfig{}
//	if err := json.Unmarshal(data, &extraConfig); err != nil {
//		syntax, ok := err.(*json.SyntaxError)
//		if !ok {
//			return nil, fmt.Errorf(
//				"Could not parse configuration: %s",
//				err)
//		}
//		return nil, newJSONParseError(data, syntax)
//	}
//	return extraConfig, nil
//}
