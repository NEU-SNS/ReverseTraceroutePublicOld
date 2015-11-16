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

//Package plcontroller is the library for creating a planet-lab controller
package plcontroller

import (
	"encoding/json"
	"net"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"gopkg.in/fsnotify.v1"
)

func (c *plControllerT) handlEvents(ec chan error) {
	log.Info("Started event handling loop")
	for {
		select {
		case <-c.shutdown:
			return
		case e := <-c.w.Events:
			if e.Op&fsnotify.Create == fsnotify.Create {
				log.Debugf("Received fs event: %v", e)
				s, err := scamper.NewSocket(
					e.Name,
					*c.config.Scamper.ConverterPath,
					json.Unmarshal,
					net.Dial)
				if err != nil {
					ec <- err
					continue
				}
				ip, err := util.IPStringToInt32(s.IP())
				if err != nil {
					ec <- err
					log.Errorf("Failed to convert socket IP: %v", err)
					continue
				}
				err = c.db.UpdateController(ip, c.ip, c.ip)
				if err != nil {
					ec <- err
					log.Errorf("Failed to update controller  %v", err)
					continue
				}
				c.client.AddSocket(s)
				break
			}
			if e.Op&fsnotify.Remove == fsnotify.Remove {
				log.Debugf("Received fs event: %v", e)
				ip := strings.Split(path.Base(e.Name), ":")[0]
				nip, err := util.IPStringToInt32(ip)
				if err != nil {
					ec <- err
					log.Errorf("Failed to convert socket IP: %v", err)
					continue
				}
				err = c.db.UpdateController(nip, 0, c.ip)
				if err != nil {
					ec <- err
					log.Errorf("Failed to update controller  %v", err)
					continue
				}
				c.client.RemoveSocket(ip)
				break
			}
		}
	}
}

//This is only for use when a server is going down
func (c *plControllerT) removeAllVps() {
	log.Debug("Removing all vps")
	for sock := range c.client.GetAllSockets() {
		ip, err := util.IPStringToInt32(sock.IP())
		if err != nil {
			continue
		}
		err = c.db.UpdateController(ip, 0, c.ip)
		if err != nil {
			log.Error(err)
		}
	}
}

func cleanDir(dir string) error {
	d, err := os.Lstat(dir)
	if err != nil {
		return nil
	}
	if err != nil {
		if err, ok := err.(*os.PathError); ok &&
			(os.IsNotExist(err.Err) || err.Err == syscall.ENOTDIR) {
			return nil
		}
		return err
	}
	if !d.IsDir() {
		return nil
	}
	dc, err := os.Open(dir)
	if err != nil {
		return err
	}
	files, err := dc.Readdirnames(-1)
	for _, fname := range files {
		err := os.RemoveAll(dir + string(os.PathSeparator) + fname)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *plControllerT) watchDir(dir string, ec chan error) {
	log.Debugf("Starting to watch dir: %s", dir)
	err := cleanDir(dir)
	if err != nil {
		log.Errorf("Failed to clean watch directory: %v", err)
		ec <- err
		return
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Failed to create watcher: %v", err)
		ec <- err
		return
	}
	c.w = w
	go c.handlEvents(ec)
	err = w.Add(dir)
	if err != nil {
		log.Errorf("Failed to add dir: %s, %v", dir, err)
		ec <- err
		return
	}
}

func (c *plControllerT) closeWatcher() {
	c.w.Close()
}
