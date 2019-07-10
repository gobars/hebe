package es

import (
	"fmt"
	"hebe/langs/goreq"
	"strings"
)

func handleCatCommand(cluster string, cmd string, options ...string) {
	body := callCatRequest(cluster, cmd, options...)
	fmt.Println(body)
}

func callCatRequest(endpoint string, api string, options ...string) string {
	uri := fmt.Sprintf("http://%s/_cat/%s?v", endpoint, api)
	if len(options) > 0 {
		uri += "&" + strings.Join(options, "&")
	}
	r := goreq.New()
	_, body, errs := r.Get(uri).End()
	if len(errs) > 0 {
		panic(errs[0])
	}
	return body
}
