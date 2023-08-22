package main

import (
	"strconv"
	"strings"

	kvcounter "kvcounter/gen"
)

const BUCKET string = "default"

type MyKVCounter struct{}

func (kv *MyKVCounter) IncrementCounter(bucket uint32, key string, amount int32) uint32 {
	incomingValue := kvcounter.WasiKeyvalueReadwriteGet(bucket, key)
	if incomingValue.IsErr() {
		return 0
	}

	b := kvcounter.WasiKeyvalueTypesIncomingValueConsumeSync(incomingValue.Unwrap())
	if b.IsErr() {
		return b.UnwrapErr()
	}

	value := string(b.Val)
	value = strings.TrimSpace(value)

	iValue, err := strconv.Atoi(value)
	if err != nil {
		return b.UnwrapErr()
	}

	outgoingValue := kvcounter.WasiKeyvalueTypesNewOutgoingValue()

	stream := kvcounter.WasiKeyvalueTypesOutgoingValueWriteBody(outgoingValue)
	if stream.IsErr() {
		return b.UnwrapErr()
	}

	inc := strconv.Itoa(iValue + int(amount))
	kvcounter.WasiIoStreamsWrite(stream.Unwrap(), []byte(inc))

	res := kvcounter.WasiKeyvalueReadwriteSet(bucket, key, outgoingValue)
	if res.IsErr() {
		return b.UnwrapErr()
	}

	stat := kvcounter.WasiKeyvalueReadwriteGet(bucket, key)
	if stat.IsErr() {
		return b.UnwrapErr()
	}

	return stat.Unwrap()
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

func (kv *MyKVCounter) Handle(request kvcounter.WasiHttpIncomingHandlerIncomingRequest, response kvcounter.WasiHttpTypesResponseOutparam) {
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

			newNum := kv.IncrementCounter(bucket.Unwrap(), "default", 1)
			writeWasiHttpResponse([]byte("New value: "+strconv.Itoa(int(newNum))), response)
		}
	default:
		return
	}

}

func init() {
	mkv := new(MyKVCounter)
	kvcounter.SetExportsWasiHttpIncomingHandler(mkv)
}

//go:generate wit-bindgen tiny-go ./wit -w kvcounter --out-dir=gen
func main() {}
