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
package etcd

import (
	etcdcl "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc/naming"
)

type watcher struct {
	wr     etcdcl.Watcher
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *watcher) Next() (*naming.Update, error) {
	for {
		resp, err := w.wr.Next(w.ctx)
		if err != nil {
			return nil, err
		}
		if resp.Node.Dir {
			continue
		}
		var act naming.OP
		if resp.Action == "set" {
			if resp.PrevNode == nil {
				act = naming.Add
			} else {
				act = naming.Modify
			}
		} else if resp.Action == "delete" {
			act = naming.Delete
		}
		if act == naming.No {
			continue
		}
		return &naming.Update{
			Op:  act,
			Key: resp.Node.Key,
			Val: resp.Node.Value,
		}, nil
	}
}

func (w *watcher) Stop() {
	w.cancel()
}

type resolver struct {
	kapi etcdcl.KeysAPI
}

func (r *resolver) NewWatcher(target string) naming.Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &watcher{
		wr:     r.kapi.Watcher(target, &etcdcl.WatcherOptions{Recursive: true}),
		ctx:    ctx,
		cancel: cancel,
	}

}

// getNode reports the naming.Update starting from node recursively.
func getNode(node *etcdcl.Node) (updates []*naming.Update) {
	for _, v := range node.Nodes {
		updates = append(updates, getNode(v)...)
	}
	if !node.Dir {
		entry := &naming.Update{
			Op:  naming.Add,
			Key: node.Key,
			Val: node.Value,
		}
		updates = []*naming.Update{entry}
	}
	return
}

func (r *resolver) Resolve(target string) ([]*naming.Update, error) {
	resp, err := r.kapi.Get(context.Background(), target, &etcdcl.GetOptions{Recursive: true})
	if err != nil {
		return nil, err
	}
	updates := getNode(resp.Node)
	return updates, nil
}

// NewResolver creates an etcd-based naming.Resolver.
func NewResolver(cfg etcdcl.Config) (naming.Resolver, error) {
	c, err := etcdcl.New(cfg)
	if err != nil {
		return nil, err
	}
	return &resolver{
		kapi: etcdcl.NewKeysAPI(c),
	}, nil
}
