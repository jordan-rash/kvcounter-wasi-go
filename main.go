package main

import (
	"strconv"
	"strings"

	kvcounter "kvcounter/gen"
)

const BUCKET string = "default"

type MyKVCounter struct{}

func (kv *MyKVCounter) IncrementCounter(bucket uint32, key string, amount int32) {
	incomingValue := kvcounter.WasiKeyvalueReadwriteGet(bucket, key)
	if incomingValue.IsErr() {
		return
	}

	b := kvcounter.WasiKeyvalueTypesIncomingValueConsumeSync(incomingValue.Unwrap())
	if b.IsErr() {
		return
	}

	value := string(b.Val)
	value = strings.TrimSpace(value)

	iValue, err := strconv.Atoi(value)
	if err != nil {
		return
	}

	outgoingValue := kvcounter.WasiKeyvalueTypesNewOutgoingValue()

	stream := kvcounter.WasiKeyvalueTypesOutgoingValueWriteBody(outgoingValue)
	if stream.IsErr() {
		return
	}

	inc := strconv.Itoa(iValue + int(amount))
	kvcounter.WasiIoStreamsWrite(stream.Unwrap(), []byte(inc))

	kvcounter.WasiKeyvalueReadwriteSet(bucket, key, outgoingValue)
}

func writeWasiHttpResponse(body []byte, responseOutparam kvcounter.WasiHttpTypesResponseOutparam) {
	headers := kvcounter.WasiHttpTypesNewFields([]kvcounter.WasiHttpTypesTuple2StringListU8TT{})

	outgoingResponse := kvcounter.WasiHttpTypesNewOutgoingResponse(200, headers)
	if outgoingResponse.IsErr() {
		return
	}

	outgoingStream := kvcounter.WasiHttpTypesOutgoingResponseWrite(outgoingResponse.Unwrap())
	if !outgoingStream.IsErr() {
		return
	}

	if kvcounter.WasiIoStreamsWrite(outgoingStream.Val, body).IsErr() {
		return
	}

	kvcounter.WasiHttpTypesFinishOutgoingStream(outgoingStream.Val)

	if kvcounter.WasiHttpTypesSetResponseOutparam(responseOutparam, outgoingResponse).IsErr() {
		return
	}
}

func (kv *MyKVCounter) Handler(request kvcounter.WasiHttpIncomingHandlerIncomingRequest, response kvcounter.WasiHttpTypesResponseOutparam) {
	method := kvcounter.WasiHttpTypesIncomingRequestMethod(request)

	pathWithQuery := kvcounter.WasiHttpTypesIncomingRequestPathWithQuery(request)
	if pathWithQuery.IsNone() {
		return
	}

	splitPath := strings.Split(pathWithQuery.Unwrap(), "?")
	trimmedPath := strings.Split(splitPath[0], "/")

	switch method.Kind() {
	case kvcounter.WasiHttpTypesMethodKindGet:
		if len(trimmedPath) > 1 && (trimmedPath[0] == "api" && trimmedPath[1] == "counter") {
			bucket := kvcounter.WasiKeyvalueTypesOpenBucket(BUCKET)
			if bucket.IsErr() {
				return
			}

			inc := kvcounter.WasiKeyvalueAtomicIncrement(bucket.Unwrap(), "default", 1)
			if inc.IsErr() {
				return
			}

			writeWasiHttpResponse([]byte("New value: "+strconv.Itoa(int(inc.Unwrap()))), response)
		}
	default:
		return
	}

}

//go:generate wit-bindgen tiny-go ./wit -w kvcounter --out-dir=gen
func main() {}
