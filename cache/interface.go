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

type Unmarshaler interface {
	Unmarshal([]byte) error
}

type CacheItem interface {
	Unmarshal(um Unmarshaler) error
	Marshal() ([]byte, error)
	Keyer
}

type Cache interface {
	Get(Keyer) (CacheItem, error)
	GetMulti([]Keyer) (map[string]CacheItem, error)
	GetVal(Keyer, Unmarshaler) error
	Set(CacheItem) error
}

type Keyer interface {
	Key() string
}

type cache struct{}

func New() Cache {
	return &cache{}
}

func (c *cache) Get(key Keyer) (CacheItem, error) {
	return nil, nil
}

func (c *cache) GetMulti(keys []Keyer) (map[string]CacheItem, error) {
	return nil, nil
}

func (c *cache) GetVal(key Keyer, marsh Unmarshaler) error {
	return nil
}

func (c *cache) Set(item CacheItem) error {
	return nil
}

/*
	Call Set -> Set calls item.CanCache()
	if it can cache -> Marshal the item and cache the data

	Call Get -> Call key.Key() try to get a key
	if a key is obtained, fetch the item from the cache.
	Otherwise return the CacheItem
*/
