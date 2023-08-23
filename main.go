package main

import (
	"embed"
	"encoding/json"
	"mime"
	"strconv"
	"strings"

	kvcounter "kvcounter/gen"
)

//go:embed ui/*
var embeddedUI embed.FS

const BUCKET string = ""

type MyKVCounter struct{}

func (kv *MyKVCounter) IncrementCounter(bucket uint32, key string, amount int32) uint32 {
	var currentValue uint32
	currentValueGet := kvcounter.WasiKeyvalueReadwriteGet(bucket, key)
	if currentValueGet.IsErr() {
		currentValue = 0
	} else {
		b := kvcounter.WasiKeyvalueTypesIncomingValueConsumeSync(currentValueGet.Unwrap())
		if b.IsErr() {
			return 100
		}
		bNum, err := strconv.Atoi(string(b.Unwrap()))
		if err != nil {
			return 101
		}
		currentValue = uint32(bNum)
	}

	newValue := currentValue + uint32(amount)
	outgoingValue := kvcounter.WasiKeyvalueTypesNewOutgoingValue()
	stream := kvcounter.WasiKeyvalueTypesOutgoingValueWriteBody(outgoingValue)
	if stream.IsErr() {
		return 102
	}

	kvcounter.WasiIoStreamsWrite(stream.Unwrap(), []byte(strconv.Itoa(int(newValue))))

	_ = kvcounter.WasiKeyvalueReadwriteSet(bucket, key, outgoingValue)
	// TODO: this is throwing an error even though it isn't erroring
	// if res.IsErr() {
	// 	return 103
	// }

	stat := kvcounter.WasiKeyvalueReadwriteGet(bucket, key)
	if stat.IsErr() {
		return 104
	}

	return newValue
}

func writeHttpResponse(responseOutparam kvcounter.WasiHttpHttpTypesResponseOutparam, statusCode uint16, inHeaders []kvcounter.WasiHttpHttpTypesTuple2StringListU8TT, body []byte) {
	headers := kvcounter.WasiHttpHttpTypesNewFields(inHeaders)

	outgoingResponse := kvcounter.WasiHttpHttpTypesNewOutgoingResponse(statusCode, headers)
	if outgoingResponse.IsErr() {
		return
	}

	outgoingStream := kvcounter.WasiHttpHttpTypesOutgoingResponseWrite(outgoingResponse.Unwrap())
	if outgoingStream.IsErr() {
		return
	}

	w := kvcounter.WasiIoStreamsWrite(outgoingStream.Val, body)
	if w.IsErr() {
		return
	}

	kvcounter.WasiHttpHttpTypesFinishOutgoingStream(outgoingStream.Val)

	outparm := kvcounter.WasiHttpHttpTypesSetResponseOutparam(responseOutparam, outgoingResponse)
	if outparm.IsErr() {
		return
	}
}

func (kv *MyKVCounter) Handle(request kvcounter.WasiHttpIncomingHandlerIncomingRequest, response kvcounter.WasiHttpHttpTypesResponseOutparam) {
	method := kvcounter.WasiHttpHttpTypesIncomingRequestMethod(request)

	pathWithQuery := kvcounter.WasiHttpHttpTypesIncomingRequestPathWithQuery(request)
	if pathWithQuery.IsNone() {
		return
	}

	splitPathQuery := strings.Split(pathWithQuery.Unwrap(), "?")

	path := splitPathQuery[0]
	trimmedPath := strings.Split(strings.TrimPrefix(path, "/"), "/")

	switch {
	case method == kvcounter.WasiHttpHttpTypesMethodGet() && len(trimmedPath) >= 2 && (trimmedPath[0] == "api" && trimmedPath[1] == "counter"):
		bucket := kvcounter.WasiKeyvalueTypesOpenBucket(BUCKET)
		if bucket.IsErr() {
			return
		}

		var inc int32 = 1
		if len(trimmedPath) == 4 {
			i, err := strconv.Atoi(trimmedPath[3])
			if err != nil {
				return
			}

			inc = int32(i)
		}

		newNum := kv.IncrementCounter(bucket.Unwrap(), "default", inc)
		resp := struct {
			Counter uint32 `json:"counter"`
		}{
			Counter: newNum,
		}

		bResp, err := json.Marshal(resp)
		if err != nil {
			return
		}

		writeHttpResponse(response, 200, []kvcounter.WasiHttpHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte("application/json")}}, bResp)
	default:
		if path == "/" {
			path = "ui/index.html"
		} else {
			path = "ui" + path
		}

		page, err := embeddedUI.ReadFile(path)
		if err != nil {
			writeHttpResponse(response, 404, []kvcounter.WasiHttpHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte("application/json")}}, []byte("{\"error\":\""+path+": not found\"}"))
		}

		ext := ""
		extSplit := strings.Split(path, ".")
		if len(extSplit) > 1 {
			ext = extSplit[len(extSplit)-1]
		}

		writeHttpResponse(response, 200, []kvcounter.WasiHttpHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte(mime.TypeByExtension(ext))}}, page)
	}

}

func init() {
	mkv := new(MyKVCounter)
	kvcounter.SetExportsWasiHttpIncomingHandler(mkv)
}

//go:generate wit-bindgen tiny-go ./wit -w kvcounter --out-dir=gen
func main() {}
