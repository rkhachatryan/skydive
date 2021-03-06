/*
 * Copyright (C) 2016 Red Hat, Inc.
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

package validator

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/gopacket/layers"
	valid "gopkg.in/validator.v2"

	"github.com/skydive-project/skydive/flow"
	ftraversal "github.com/skydive-project/skydive/flow/traversal"
	"github.com/skydive-project/skydive/topology"
	"github.com/skydive-project/skydive/topology/graph"
	"github.com/skydive-project/skydive/topology/graph/traversal"
)

// Validator interface used to validate value type in Gremlin expression
type Validator interface {
	Validate() error
}

var (
	skydiveValidator = valid.NewValidator()

	//IPNotValid validator
	IPNotValid = func() error {
		return valid.TextErr{Err: errors.New("Not a IP addr")}
	}
	// GremlinNotValid validator
	GremlinNotValid = func(err error) error {
		return valid.TextErr{Err: fmt.Errorf("Not a valid Gremlin expression: %s", err.Error())}
	}
	// BPFFilterNotValid validator
	BPFFilterNotValid = func(err error) error {
		return valid.TextErr{Err: fmt.Errorf("Not a valid BPF expression: %s", err.Error())}
	}
	// CaptureHeaderSizeNotValid validator
	CaptureHeaderSizeNotValid = func(min, max uint32) error {
		return valid.TextErr{Err: fmt.Errorf("A valid header size is >= %d && <= %d", min, max)}
	}
	// RawPacketLimitNotValid validator
	RawPacketLimitNotValid = func(min, max uint32) error {
		return valid.TextErr{Err: fmt.Errorf("A valid raw packet limit size is > %d && <= %d", min, max)}
	}
)

func isIP(v interface{}, param string) error {
	ip, ok := v.(string)
	if !ok {
		return IPNotValid()
	}
	/* Parse/Check IPv4 and IPv6 address */
	if n := net.ParseIP(ip); n == nil {
		return IPNotValid()
	}
	return nil
}

func isGremlinExpr(v interface{}, param string) error {
	query, ok := v.(string)
	if !ok {
		return GremlinNotValid(errors.New("not a string"))
	}

	tr := traversal.NewGremlinTraversalParser(&graph.Graph{})
	tr.AddTraversalExtension(topology.NewTopologyTraversalExtension())
	tr.AddTraversalExtension(ftraversal.NewFlowTraversalExtension(nil, nil))

	if _, err := tr.Parse(strings.NewReader(query), false); err != nil {
		return GremlinNotValid(err)
	}

	return nil
}

func isBPFFilter(v interface{}, param string) error {
	bpfFilter, ok := v.(string)
	if !ok {
		return BPFFilterNotValid(errors.New("not a string"))
	}

	if _, err := flow.BPFFilterToRaw(layers.LinkTypeEthernet, flow.MaxCaptureLength, bpfFilter); err != nil {
		return BPFFilterNotValid(err)
	}

	return nil
}

func isValidCaptureHeaderSize(v interface{}, param string) error {
	headerSize, ok := v.(int)
	hs := uint32(headerSize)
	if !ok || hs < 0 || (hs > 0 && hs < 14) || hs > flow.MaxCaptureLength {
		return CaptureHeaderSizeNotValid(14, flow.MaxCaptureLength)
	}

	return nil
}

func isValidRawPacketLimit(v interface{}, param string) error {
	limit, ok := v.(int)
	l := uint32(limit)
	// The current limitation of 10 packet could be removed once flow will be
	// transfered over tcp.
	if !ok || l < 0 || l > flow.MaxRawPacketLimit {
		return RawPacketLimitNotValid(0, flow.MaxRawPacketLimit)
	}

	return nil
}

// Validate an object based on previously (at init) registered function
func Validate(value interface{}) error {
	if err := skydiveValidator.Validate(value); err != nil {
		return err
	}

	if obj, ok := value.(Validator); ok {
		return obj.Validate()
	}

	return nil
}

func init() {
	skydiveValidator.SetValidationFunc("isIP", isIP)
	skydiveValidator.SetValidationFunc("isGremlinExpr", isGremlinExpr)
	skydiveValidator.SetValidationFunc("isBPFFilter", isBPFFilter)
	skydiveValidator.SetValidationFunc("isValidCaptureHeaderSize", isValidCaptureHeaderSize)
	skydiveValidator.SetValidationFunc("isValidRawPacketLimit", isValidRawPacketLimit)
	skydiveValidator.SetTag("valid")
}
