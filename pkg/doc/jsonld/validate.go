/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package jsonld

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/piprate/json-gold/ld"

	"github.com/hyperledger/aries-framework-go/pkg/doc/signature/jsonld"
	"github.com/hyperledger/aries-framework-go/pkg/doc/util/json"
)

type validateOpts struct {
	strict               bool
	jsonldDocumentLoader ld.DocumentLoader
	externalContext      []string
}

// ValidateOpts sets jsonld validation options.
type ValidateOpts func(opts *validateOpts)

// WithDocumentLoader option is for passing custom JSON-LD document loader.
func WithDocumentLoader(jsonldDocumentLoader ld.DocumentLoader) ValidateOpts {
	return func(opts *validateOpts) {
		opts.jsonldDocumentLoader = jsonldDocumentLoader
	}
}

// WithExternalContext option is for definition of external context when doing JSON-LD operations.
func WithExternalContext(externalContext []string) ValidateOpts {
	return func(opts *validateOpts) {
		opts.externalContext = externalContext
	}
}

// WithStrictValidation sets if strict validation should be used.
func WithStrictValidation(checkStructure bool) ValidateOpts {
	return func(opts *validateOpts) {
		opts.strict = checkStructure
	}
}

func getValidateOpts(options []ValidateOpts) *validateOpts {
	result := &validateOpts{
		strict: true,
	}

	for _, opt := range options {
		opt(result)
	}

	return result
}

// ValidateJSONLD validates jsonld structure.
func ValidateJSONLD(doc string, options ...ValidateOpts) error {
	opts := getValidateOpts(options)

	docMap, err := json.ToMap(doc)
	if err != nil {
		return fmt.Errorf("convert JSON-LD doc to map: %w", err)
	}

	jsonldProc := jsonld.Default()

	docCompactedMap, err := jsonldProc.Compact(docMap,
		nil, jsonld.WithDocumentLoader(opts.jsonldDocumentLoader),
		jsonld.WithExternalContext(opts.externalContext...))
	if err != nil {
		return fmt.Errorf("compact JSON-LD document: %w", err)
	}

	if opts.strict && !mapsHaveSameStructure(docMap, docCompactedMap) {
		return errors.New("JSON-LD doc has different structure after compaction")
	}

	return nil
}

func mapsHaveSameStructure(originalMap, compactedMap map[string]interface{}) bool {
	original := compactMap(originalMap)
	compacted := compactMap(compactedMap)

	if reflect.DeepEqual(original, compacted) {
		return true
	}

	if len(original) != len(compacted) {
		return false
	}

	for k, v1 := range original {
		v1Map, isMap := v1.(map[string]interface{})
		if !isMap {
			continue
		}

		v2, present := compacted[k]
		if !present { // special case - the name of the map was mapped, cannot guess what's a new name
			continue
		}

		v2Map, isMap := v2.(map[string]interface{})
		if !isMap {
			return false
		}

		if !mapsHaveSameStructure(v1Map, v2Map) {
			return false
		}
	}

	return true
}

func compactMap(m map[string]interface{}) map[string]interface{} {
	mCopy := make(map[string]interface{})

	for k, v := range m {
		// ignore context
		if k == "@context" {
			continue
		}

		vNorm := compactValue(v)

		switch kv := vNorm.(type) {
		case []interface{}:
			mCopy[k] = compactSlice(kv)

		case map[string]interface{}:
			mCopy[k] = compactMap(kv)

		default:
			mCopy[k] = vNorm
		}
	}

	return mCopy
}

func compactSlice(s []interface{}) []interface{} {
	sCopy := make([]interface{}, len(s))

	for i := range s {
		sItem := compactValue(s[i])

		switch sItem := sItem.(type) {
		case map[string]interface{}:
			sCopy[i] = compactMap(sItem)

		default:
			sCopy[i] = sItem
		}
	}

	return sCopy
}

func compactValue(v interface{}) interface{} {
	switch cv := v.(type) {
	case []interface{}:
		// consists of only one element
		if len(cv) == 1 {
			return compactValue(cv[0])
		}

		return cv

	case map[string]interface{}:
		// contains "id" element only
		if len(cv) == 1 {
			if _, ok := cv["id"]; ok {
				return cv["id"]
			}
		}

		return cv

	default:
		return cv
	}
}