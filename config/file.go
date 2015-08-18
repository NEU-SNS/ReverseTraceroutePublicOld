/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"

	"github.com/golang/glog"

	"gopkg.in/yaml.v2"
)

func parseYamlConfig(path string, opts interface{}) error {
	f, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		glog.Infof("Error opening config file: %s, %v", path, err)
		var ret error
		if os.IsNotExist(err) {
			// Return nil, not a problem if we don't find a config file
			ret = nil
		} else {
			ret = err
		}
		return ret
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, opts)
	if err != nil {
		return err
	}

	return nil
}

var configPaths = make(map[string]configPath)

type configPath struct {
	Path  string
	Order int
}

var lastConfig int

// AddConfigPath added the path to possible config files
func AddConfigPath(path string) {
	order := lastConfig
	lastConfig++
	cp := configPath{Path: path, Order: order}
	configPaths[path] = cp
}

type configPathOrder []configPath

func (cp configPathOrder) Len() int           { return len(cp) }
func (cp configPathOrder) Swap(i, j int)      { cp[i], cp[j] = cp[j], cp[i] }
func (cp configPathOrder) Less(i, j int) bool { return cp[i].Order < cp[j].Order }

func mergeFiles(f *flag.FlagSet, opts interface{}) error {
	ov := reflect.ValueOf(opts)
	if ov.Kind() != reflect.Ptr || ov.IsNil() {
		return fmt.Errorf("mergeFiles, opts invalid type")
	}
	paths := make([]configPath, len(configPaths))
	var i int
	for _, val := range configPaths {
		paths[i] = val
		i++
	}
	sort.Sort(configPathOrder(paths))
	for _, path := range paths {
		err := parseYamlConfig(path.Path, opts)
		if err != nil {
			return err
		}
		ops, err := buildMap(ov)
		if err != nil {
			glog.Errorf("Failed to build map: %v", err)
			return nil
		}
		err = handleFile(f, ops)
		if err != nil {
			return err
		}
	}
	return nil
}

func mergeMaps(a, b map[string]string) {
	if b == nil {
		return
	}
	for key, value := range b {
		a[key] = value
	}
}

func buildMap(opts reflect.Value) (map[string]string, error) {
	res := make(map[string]string)
	ot := opts.Elem().Type()
	numFields := ot.NumField()
	for i := 0; i < numFields; i++ {
		field := ot.Field(i)
		if field.Type.Kind() == reflect.Struct {
			subOpts, err := buildMap(opts.Elem().Field(i).Addr())
			if err != nil {
				return nil, err
			}
			mergeMaps(res, subOpts)
			continue
		}
		name := field.Tag.Get("flag")
		if name == "" {
			continue
		}
		if opts.Elem().Field(i).IsNil() {
			continue
		}
		res[name] = fmt.Sprintf("%v", opts.Elem().Field(i).Elem())
	}
	return res, nil
}

func handleFile(f *flag.FlagSet, opts map[string]string) error {
	err := merge(f, func(name string) *string {
		if val, ok := opts[name]; ok {
			if val != "" {
				return &val
			}
		}
		return nil
	})
	return err
}
