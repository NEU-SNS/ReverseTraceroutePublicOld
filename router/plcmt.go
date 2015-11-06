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

package router

import (
	"fmt"
	"io"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type plmt struct {
	s  ServiceDef
	c  *grpc.ClientConn
	cl plcontrollerapi.PLControllerClient
}

func (p plmt) Ping(ctx con.Context, pa *dm.PingArg) (<-chan *dm.Ping, error) {
	ret := make(chan *dm.Ping)
	ps, err := p.cl.Ping(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if err := ps.Send(pa); err != nil {
		log.Error(err)
	}
	go func() {
		for {
			in, err := ps.Recv()
			if err == io.EOF {
				close(ret)
				return
			}
			if err != nil {
				log.Error(err)
				return
			}
			ret <- in
		}
	}()
	ps.CloseSend()
	return ret, nil
}

func (p plmt) Traceroute(ctx con.Context, t *dm.TracerouteArg) (<-chan *dm.Traceroute, error) {
	ret := make(chan *dm.Traceroute)
	ps, err := p.cl.Traceroute(ctx)
	if err != nil {
		return nil, err
	}
	if err := ps.Send(t); err != nil {
		log.Error(err)
	}
	go func() {
		for {
			in, err := ps.Recv()
			if err == io.EOF {
				close(ret)
				return
			}
			if err != nil {
				log.Error(err)
				return
			}
			ret <- in
		}
	}()
	ps.CloseSend()
	return ret, nil
}

func (p plmt) GetVPs(ctx con.Context, v *dm.VPRequest) (<-chan *dm.VPReturn, error) {
	ret := make(chan *dm.VPReturn)
	ps, err := p.cl.GetVPs(ctx, v)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	go func() {
		for {
			in, err := ps.Recv()
			if err == io.EOF {
				close(ret)
				return
			}
			if err != nil {
				log.Error(err)
				continue
			}
			ret <- in
		}
	}()
	return ret, nil
}

func (p plmt) Close() error {
	return p.c.Close()
}

func createPLMT(s ServiceDef) (plmt, error) {
	var ret plmt
	opts := make([]grpc.DialOption, 1)
	opts[0] = grpc.WithInsecure()
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", s.Addr, s.Port), opts...)
	if err != nil {
		log.Error(err)
		return ret, err
	}
	cl := plcontrollerapi.NewPLControllerClient(conn)
	ret.c = conn
	ret.cl = cl
	ret.s = s
	return ret, nil
}
