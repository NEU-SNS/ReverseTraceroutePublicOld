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

package server_test

import (
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/mocks"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/repo"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/server"
	"github.com/NEU-SNS/ReverseTraceroute/cache"
	cmocks "github.com/NEU-SNS/ReverseTraceroute/controller/mocks"
	vpmocks "github.com/NEU-SNS/ReverseTraceroute/vpservice/mocks"
	vppb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/stretchr/testify/mock"
)

type mockCache struct {
	cache map[string][]byte
}

type mockItem struct {
	key string
	val []byte
}

func (mi *mockItem) Key() string {
	return mi.key
}

func (mi *mockItem) Value() []byte {
	return mi.val
}

func (mc *mockCache) Get(key string) (cache.Item, error) {
	if mc.cache == nil {
		mc.cache = make(map[string][]byte)
	}
	if val, ok := mc.cache[key]; ok {
		return &mockItem{key: key, val: val}, nil
	}
	return nil, cache.ErrorCacheMiss
}

func (mc *mockCache) GetMulti(keys []string) (map[string]cache.Item, error) {
	panic("unimplemented")
}
func (mc *mockCache) Set(key string, val []byte) error {
	if mc.cache == nil {
		mc.cache = make(map[string][]byte)
	}
	mc.cache[key] = val
	return nil
}

func (mc *mockCache) SetWithExpire(string, []byte, int32) error {
	panic("unimplemented")
}

func TestGetPathsWithToken(t *testing.T) {
	trsm := &mocks.TRStore{}
	trsm.On("FindIntersectingTraceroute",
		mock.AnythingOfType("types.IntersectionQuery")).Return(nil, repo.ErrNoIntFound)
	trsm.On("GetAtlasSources",
		mock.AnythingOfType("uint32"), mock.AnythingOfType("time.Duration")).Return([]uint32{}, nil)
	clm := &cmocks.Client{}
	vpsm := &vpmocks.VPSource{}
	vpsm.On("GetVPs").Return(&vppb.VPReturn{}, nil)
	var opts []server.Option
	opts = append(opts, server.WithClient(clm),
		server.WithTRS(trsm), server.WithVPS(vpsm),
		server.WithCache(&mockCache{}))

	serv := server.NewServer(opts...)
	load := []*pb.IntersectionRequest{
		&pb.IntersectionRequest{
			Address: 9,
			Dest:    10,
			Src:     2,
		},
		&pb.IntersectionRequest{
			Address: 0,
			Dest:    1,
			Src:     2,
		},
		&pb.IntersectionRequest{
			Address: 11,
			Dest:    15,
			Src:     2,
		},
		&pb.IntersectionRequest{
			Address: 19,
			Dest:    20,
			Src:     2,
		},
	}
	var responses []*pb.IntersectionResponse
	for _, l := range load {
		res, err := serv.GetIntersectingPath(l)
		if err != nil {
			t.Fatal("Failed to load requests ", err)
		}
		if res.Type != pb.IResponseType_TOKEN {
			t.Fatalf("GetIntersectingPath(%v) expected token response got(%v)", l, res)
		}
		responses = append(responses, res)
	}
	// This is ugly, but wait so that other goroutines can run
	<-time.After(time.Second * 2)
	for _, resp := range responses {
		r, err := serv.GetPathsWithToken(&pb.TokenRequest{
			Token: resp.Token,
		})
		if err != nil {
			t.Fatalf("Failed to get path with token(%v) got err %v", resp.Token, err)
		}
		if r.Token != resp.Token {
			t.Fatalf("Expected Token[%v], Got Token[%v]", resp.Token, r.Token)
		}
		if r.Type != pb.IResponseType_NONE_FOUND {
			t.Fatalf("Expected[%v], Got[%v]", pb.IResponseType_NONE_FOUND, r.Type)
		}
	}
}

func TestGetPathsWithTokenInvalidToken(t *testing.T) {
	trsm := &mocks.TRStore{}
	trsm.On("FindIntersectingTraceroute",
		mock.AnythingOfType("types.IntersectionQuery")).Return(nil, repo.ErrNoIntFound)
	clm := &cmocks.Client{}
	vpsm := &vpmocks.VPSource{}
	var opts []server.Option
	opts = append(opts, server.WithClient(clm),
		server.WithTRS(trsm), server.WithVPS(vpsm),
		server.WithCache(&mockCache{}))
	serv := server.NewServer(opts...)
	tokenReq := &pb.TokenRequest{
		Token: 99999,
	}
	r, err := serv.GetPathsWithToken(tokenReq)
	if err != nil {
		t.Fatalf("Expected nil error received resp[%v], err[%v]", r, err)
	}
	if r.Type != pb.IResponseType_ERROR || r.Token != tokenReq.Token {
		t.Fatalf("Unexpected response resp[%v], err[%v]", r, err)
	}
}
