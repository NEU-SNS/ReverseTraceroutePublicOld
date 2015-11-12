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

package cache

import (
	"fmt"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
)

var (
	ErrorNoClient     = fmt.Errorf("No cache client!")
	ErrorCacheMiss    = memcache.ErrCacheMiss
	ErrorNotStored    = memcache.ErrNotStored
	ErrorServerError  = memcache.ErrServerError
	ErrorMalformedKey = memcache.ErrMalformedKey
	ErrorNoServers    = memcache.ErrNoServers
)

func toError(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case memcache.ErrCacheMiss:
		return ErrorCacheMiss
	case memcache.ErrNotStored:
		return ErrorNotStored
	case memcache.ErrServerError:
		return ErrorServerError
	case memcache.ErrMalformedKey:
		return ErrorMalformedKey
	case memcache.ErrNoServers:
		return ErrorNoServers
	default:
		return err
	}

}

type outItem struct {
	data []byte
	key  string
}

func (o outItem) Key() string {
	return o.key
}

func (o outItem) Value() []byte {
	return o.data
}

func fixKey(in string) string {
	res := strings.SplitAfterN(in, "_", 2)
	return res[1]
}

func toOutItem(key string, data []byte) outItem {

	return outItem{
		data: data,
		key:  fixKey(key),
	}
}

type cache struct {
	c      *memcache.Client
	prefix string
}

// New creates a new cache
func New(servers ServerList) Cache {
	return &cache{
		c:      memcache.New(servers...),
		prefix: "DEFAULT",
	}
}

func (c *cache) SetPrefix(pre string) {
	c.prefix = pre
}

func makeKey(pre, val string) string {
	return fmt.Sprintf("%s_%s", pre, val)
}

func (c *cache) Get(key string) (Item, error) {
	if c.c == nil {
		return nil, ErrorNoClient
	}
	item, err := c.c.Get(makeKey(c.prefix, key))
	if err != nil {
		return nil, toError(err)
	}
	return toOutItem(item.Key, item.Value), nil
}

func (c *cache) GetMulti(keys []string) (map[string]Item, error) {
	if c.c == nil {
		return nil, ErrorNoClient
	}
	nkeys := len(keys)
	ukeys := make([]string, nkeys)
	for i, key := range keys {
		ukeys[i] = makeKey(c.prefix, key)
	}
	multi, err := c.c.GetMulti(ukeys)
	if err != nil {
		return nil, toError(err)
	}
	ret := make(map[string]Item)
	for k, v := range multi {
		ret[fixKey(k)] = toOutItem(v.Key, v.Value)
	}
	return ret, nil
}

func (c *cache) Set(key string, val []byte) error {
	if c.c == nil {
		return ErrorNoClient
	}
	// Default to 15 minute expire. This is set
	// based on the old system, may need to be
	// updated in the future
	return c.SetWithExpire(key, val, 15*60)
}

func (c *cache) SetWithExpire(key string, val []byte, exp int32) error {
	if c.c == nil {
		return ErrorNoClient
	}
	return toError(c.c.Set(&memcache.Item{
		Key:        makeKey(c.prefix, key),
		Value:      val,
		Expiration: exp,
	}))
}
