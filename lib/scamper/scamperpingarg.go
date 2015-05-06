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
package scamper

var pingArg = map[string]option{
	"Dst": option{
		format: "%s",
		opt:    stringOpt,
	},
	"Spoof": option{
		format: "-O spoof",
		opt:    boolOpt,
	},
	"SAddr": option{
		format: "-S %s",
		opt:    stringOpt,
	},
	"RR": option{
		format: "-RR",
		opt:    boolOpt,
	},
	"Payload": option{
		format: "-B %s",
		opt:    stringOpt,
	},
	"Count": option{
		format: "-c %s",
		opt:    stringOpt,
	},
	"IcmpSum": option{
		format: "-C %s",
		opt:    stringOpt,
	},
	"DPort": option{
		format: "-d %s",
		opt:    stringOpt,
	},
	"SPort": option{
		format: "-F %s",
		opt:    stringOpt,
	},
	"Wait": option{
		format: "-i %s",
		opt:    stringOpt,
	},
	"Ttl": option{
		format: "-m %s",
		opt:    stringOpt,
	},
	"Mtu": option{
		format: "-M %s",
		opt:    stringOpt,
	},
	"ReplyCount": option{
		format: "-o %s",
		opt:    stringOpt,
	},
	"Pattern": option{
		format: "-p %s",
		opt:    stringOpt,
	},
	"Method": option{
		format: "-P %s",
		opt:    stringOpt,
	},
	"Size": option{
		format: "-s %s",
		opt:    stringOpt,
	},
	"UserId": option{
		format: "-U %s",
		opt:    stringOpt,
	},
	"Tos": option{
		format: "-z %s",
		opt:    stringOpt,
	},
	"TimeStamp": option{
		format: "-T %s",
		opt:    stringOpt,
	},
}
