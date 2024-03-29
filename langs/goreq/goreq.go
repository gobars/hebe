// Package gorequest inspired by Nodejs SuperAgent provides easy-way to write http client
package goreq

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/moul/http2curl"

	"golang.org/x/net/publicsuffix"
)

type Request *http.Request
type Response *http.Response

// HTTP methods we support
const (
	POST    = "POST"
	GET     = "GET"
	HEAD    = "HEAD"
	PUT     = "PUT"
	DELETE  = "DELETE"
	PATCH   = "PATCH"
	OPTIONS = "OPTIONS"
)

// A SuperAgent is a object storing all request data for client.
type SuperAgent struct {
	Url               string
	Method            string
	Header            map[string]string
	TargetType        string
	ForceType         string
	Data              map[string]interface{}
	SliceData         []interface{}
	FormData          url.Values
	QueryData         url.Values
	BounceToRawString bool
	RawString         string
	Client            *http.Client
	Transport         *http.Transport
	Cookies           []*http.Cookie
	Errors            []error
	BasicAuth         struct{ Username, Password string }
	Debug             bool
	CurlCommand       bool
	logger            *log.Logger
}

var DisableTransportSwap = false

// Used to create a new SuperAgent object.
func New() *SuperAgent {
	cookiejarOptions := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, _ := cookiejar.New(&cookiejarOptions)
	s := &SuperAgent{
		TargetType:        "json",
		Data:              make(map[string]interface{}),
		Header:            make(map[string]string),
		RawString:         "",
		SliceData:         []interface{}{},
		FormData:          url.Values{},
		QueryData:         url.Values{},
		BounceToRawString: false,
		Client:            &http.Client{Jar: jar},
		Transport:         &http.Transport{},
		Cookies:           make([]*http.Cookie, 0),
		Errors:            nil,
		BasicAuth:         struct{ Username, Password string }{},
		Debug:             false,
		CurlCommand:       false,
		logger:            log.New(os.Stderr, "[gorequest]", log.LstdFlags),
	}
	return s
}

// Enable the debug mode which logs request/response detail
func (s *SuperAgent) SetDebug(enable bool) *SuperAgent {
	s.Debug = enable
	return s
}

// Enable the curlcommand mode which display a CURL command line
func (s *SuperAgent) SetCurlCommand(enable bool) *SuperAgent {
	s.CurlCommand = enable
	return s
}

func (s *SuperAgent) SetLogger(logger *log.Logger) *SuperAgent {
	s.logger = logger
	return s
}

// Clear SuperAgent data for another new request.
func (s *SuperAgent) ClearSuperAgent() {
	s.Url = ""
	s.Method = ""
	s.Header = make(map[string]string)
	s.Data = make(map[string]interface{})
	s.SliceData = []interface{}{}
	s.FormData = url.Values{}
	s.QueryData = url.Values{}
	s.BounceToRawString = false
	s.RawString = ""
	s.ForceType = ""
	s.TargetType = "json"
	s.Cookies = make([]*http.Cookie, 0)
	s.Errors = nil
}

// Just a wrapper to initialize SuperAgent instance by method string
func (s *SuperAgent) CustomMethod(method, targetUrl string) *SuperAgent {
	switch method {
	case POST:
		return s.Post(targetUrl)
	case GET:
		return s.Get(targetUrl)
	case HEAD:
		return s.Head(targetUrl)
	case PUT:
		return s.Put(targetUrl)
	case DELETE:
		return s.Delete(targetUrl)
	case PATCH:
		return s.Patch(targetUrl)
	case OPTIONS:
		return s.Options(targetUrl)
	default:
		s.ClearSuperAgent()
		s.Method = method
		s.Url = targetUrl
		s.Errors = nil
		return s
	}
}

func (s *SuperAgent) Get(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = GET
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Post(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = POST
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Head(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = HEAD
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Put(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = PUT
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Delete(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = DELETE
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Patch(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = PATCH
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Options(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = OPTIONS
	s.Url = targetUrl
	s.Errors = nil
	return s
}

// Set is used for setting header fields.
// Example. To set `Accept` as `application/json`
//
//    gorequest.New().
//      Post("/gamelist").
//      Set("Accept", "application/json").
//      End()
func (s *SuperAgent) Set(param string, value string) *SuperAgent {
	s.Header[param] = value
	return s
}

// SetBasicAuth sets the basic authentication header
// Example. To set the header for username "myuser" and password "mypass"
//
//    gorequest.New()
//      Post("/gamelist").
//      SetBasicAuth("myuser", "mypass").
//      End()
func (s *SuperAgent) SetBasicAuth(username string, password string) *SuperAgent {
	s.BasicAuth = struct{ Username, Password string }{username, password}
	return s
}

// AddCookie adds a cookie to the request. The behavior is the same as AddCookie on Request from net/http
func (s *SuperAgent) AddCookie(c *http.Cookie) *SuperAgent {
	s.Cookies = append(s.Cookies, c)
	return s
}

// AddCookies is a convenient method to add multiple cookies
func (s *SuperAgent) AddCookies(cookies []*http.Cookie) *SuperAgent {
	s.Cookies = append(s.Cookies, cookies...)
	return s
}

var Types = map[string]string{
	"html":       "text/html",
	"json":       "application/json",
	"xml":        "application/xml",
	"text":       "text/plain",
	"urlencoded": "application/x-www-form-urlencoded",
	"form":       "application/x-www-form-urlencoded",
	"form-data":  "application/x-www-form-urlencoded",
}

// Type is a convenience function to specify the data type to send.
// For example, to send data as `application/x-www-form-urlencoded` :
//
//    gorequest.New().
//      Post("/recipe").
//      Type("form").
//      Send(`{ "name": "egg benedict", "category": "brunch" }`).
//      End()
//
// This will POST the body "name=egg benedict&category=brunch" to url /recipe
//
// GoRequest supports
//
//    "text/html" uses "html"
//    "application/json" uses "json"
//    "application/xml" uses "xml"
//    "text/plain" uses "text"
//    "application/x-www-form-urlencoded" uses "urlencoded", "form" or "form-data"
//
func (s *SuperAgent) Type(typeStr string) *SuperAgent {
	if _, ok := Types[typeStr]; ok {
		s.ForceType = typeStr
	} else {
		s.Errors = append(s.Errors, errors.New("Type func: incorrect type \""+typeStr+"\""))
	}
	return s
}

// Query function accepts either json string or strings which will form a query-string in url of GET method or body of POST method.
// For example, making "/search?query=bicycle&size=50x50&weight=20kg" using GET method:
//
//      gorequest.New().
//        Get("/search").
//        Query(`{ query: 'bicycle' }`).
//        Query(`{ size: '50x50' }`).
//        Query(`{ weight: '20kg' }`).
//        End()
//
// Or you can put multiple json values:
//
//      gorequest.New().
//        Get("/search").
//        Query(`{ query: 'bicycle', size: '50x50', weight: '20kg' }`).
//        End()
//
// Strings are also acceptable:
//
//      gorequest.New().
//        Get("/search").
//        Query("query=bicycle&size=50x50").
//        Query("weight=20kg").
//        End()
//
// Or even Mixed! :)
//
//      gorequest.New().
//        Get("/search").
//        Query("query=bicycle").
//        Query(`{ size: '50x50', weight:'20kg' }`).
//        End()
//
func (s *SuperAgent) Query(content interface{}) *SuperAgent {
	switch v := reflect.ValueOf(content); v.Kind() {
	case reflect.String:
		s.queryString(v.String())
	case reflect.Struct:
		s.queryStruct(v.Interface())
	default:
	}
	return s
}

func (s *SuperAgent) queryStruct(content interface{}) *SuperAgent {
	if marshalContent, err := json.Marshal(content); err != nil {
		s.Errors = append(s.Errors, err)
	} else {
		var val map[string]interface{}
		if err := json.Unmarshal(marshalContent, &val); err != nil {
			s.Errors = append(s.Errors, err)
		} else {
			for k, v := range val {
				k = strings.ToLower(k)
				s.QueryData.Add(k, v.(string))
			}
		}
	}
	return s
}

func (s *SuperAgent) queryString(content string) *SuperAgent {
	var val map[string]string
	if err := json.Unmarshal([]byte(content), &val); err == nil {
		for k, v := range val {
			s.QueryData.Add(k, v)
		}
	} else {
		if queryVal, err := url.ParseQuery(content); err == nil {
			for k, _ := range queryVal {
				s.QueryData.Add(k, queryVal.Get(k))
			}
		} else {
			s.Errors = append(s.Errors, err)
		}
		// TODO: need to check correct format of 'field=val&field=val&...'
	}
	return s
}

// As Go conventions accepts ; as a synonym for &. (https://github.com/golang/go/issues/2210)
// Thus, Query won't accept ; in a querystring if we provide something like fields=f1;f2;f3
// This Param is then created as an alternative method to solve this.
func (s *SuperAgent) Param(key string, value string) *SuperAgent {
	s.QueryData.Add(key, value)
	return s
}

func (s *SuperAgent) Timeout(timeout time.Duration) *SuperAgent {
	s.Transport.Dial = func(network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(network, addr, timeout)
		if err != nil {
			s.Errors = append(s.Errors, err)
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(timeout))
		return conn, nil
	}
	return s
}

// Set TLSClientConfig for underling Transport.
// One example is you can use it to disable security check (https):
//
//      gorequest.New().TLSClientConfig(&tls.Config{ InsecureSkipVerify: true}).
//        Get("https://disable-security-check.com").
//        End()
//
func (s *SuperAgent) TLSClientConfig(config *tls.Config) *SuperAgent {
	s.Transport.TLSClientConfig = config
	return s
}

// Proxy function accepts a proxy url string to setup proxy url for any request.
// It provides a convenience way to setup proxy which have advantages over usual old ways.
// One example is you might try to set `http_proxy` environment. This means you are setting proxy up for all the requests.
// You will not be able to send different request with different proxy unless you change your `http_proxy` environment again.
// Another example is using Golang proxy setting. This is normal prefer way to do but too verbase compared to GoRequest's Proxy:
//
//      gorequest.New().Proxy("http://myproxy:9999").
//        Post("http://www.google.com").
//        End()
//
// To set no_proxy, just put empty string to Proxy func:
//
//      gorequest.New().Proxy("").
//        Post("http://www.google.com").
//        End()
//
func (s *SuperAgent) Proxy(proxyUrl string) *SuperAgent {
	parsedProxyUrl, err := url.Parse(proxyUrl)
	if err != nil {
		s.Errors = append(s.Errors, err)
	} else if proxyUrl == "" {
		s.Transport.Proxy = nil
	} else {
		s.Transport.Proxy = http.ProxyURL(parsedProxyUrl)
	}
	return s
}

func (s *SuperAgent) RedirectPolicy(policy func(req Request, via []Request) error) *SuperAgent {
	s.Client.CheckRedirect = func(r *http.Request, v []*http.Request) error {
		vv := make([]Request, len(v))
		for i, r := range v {
			vv[i] = Request(r)
		}
		return policy(Request(r), vv)
	}
	return s
}

// Send function accepts either json string or query strings which is usually used to assign data to POST or PUT method.
// Without specifying any type, if you give Send with json data, you are doing requesting in json format:
//
//      gorequest.New().
//        Post("/search").
//        Send(`{ query: 'sushi' }`).
//        End()
//
// While if you use at least one of querystring, GoRequest understands and automatically set the Content-Type to `application/x-www-form-urlencoded`
//
//      gorequest.New().
//        Post("/search").
//        Send("query=tonkatsu").
//        End()
//
// So, if you want to strictly send json format, you need to use Type func to set it as `json` (Please see more details in Type function).
// You can also do multiple chain of Send:
//
//      gorequest.New().
//        Post("/search").
//        Send("query=bicycle&size=50x50").
//        Send(`{ wheel: '4'}`).
//        End()
//
// From v0.2.0, Send function provide another convenience way to work with Struct type. You can mix and match it with json and query string:
//
//      type BrowserVersionSupport struct {
//        Chrome string
//        Firefox string
//      }
//      ver := BrowserVersionSupport{ Chrome: "37.0.2041.6", Firefox: "30.0" }
//      gorequest.New().
//        Post("/update_version").
//        Send(ver).
//        Send(`{"Safari":"5.1.10"}`).
//        End()
//
// If you have set Type to text or Content-Type to text/plain, content will be sent as raw string in body instead of form
//
//      gorequest.New().
//        Post("/greet").
//        Type("text").
//        Send("hello world").
//        End()
//
func (s *SuperAgent) Send(content interface{}) *SuperAgent {
	// TODO: add normal text mode or other mode to Send func
	switch v := reflect.ValueOf(content); v.Kind() {
	case reflect.String:
		s.SendString(v.String())
	case reflect.Struct:
		s.SendStruct(v.Interface())
	case reflect.Slice:
		// change to slice
		slice := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			slice[i] = v.Index(i).Interface()
		}
		s.SendSlice(slice)
	default:
		// TODO: leave default for handling other types in the future such as number, byte, etc...
		// TODO: Add support for slice and array
	}
	return s
}

// SendSlice (similar to SendString) returns SuperAgent's itself for any next chain and takes content []interface{} as a parameter.
// Its duty is to append slice of interface{} into s.SliceData ([]interface{}) which later changes into json array in the End() func.
func (s *SuperAgent) SendSlice(content []interface{}) *SuperAgent {
	s.SliceData = append(s.SliceData, content...)
	return s
}

// SendStruct (similar to SendString) returns SuperAgent's itself for any next chain and takes content interface{} as a parameter.
// Its duty is to transfrom interface{} (implicitly always a struct) into s.Data (map[string]interface{}) which later changes into appropriate format such as json, form, text, etc. in the End() func.
func (s *SuperAgent) SendStruct(content interface{}) *SuperAgent {
	if marshalContent, err := json.Marshal(content); err != nil {
		s.Errors = append(s.Errors, err)
	} else {
		var val map[string]interface{}
		d := json.NewDecoder(bytes.NewBuffer(marshalContent))
		d.UseNumber()
		if err := d.Decode(&val); err != nil {
			s.Errors = append(s.Errors, err)
		} else {
			for k, v := range val {
				s.Data[k] = v
			}
		}
	}
	return s
}

// SendString returns SuperAgent's itself for any next chain and takes content string as a parameter.
// Its duty is to transform String into s.Data (map[string]interface{}) which later changes into appropriate format such as json, form, text, etc. in the End func.
// Send implicitly uses SendString and you should use Send instead of this.
func (s *SuperAgent) SendString(content string) *SuperAgent {
	if !s.BounceToRawString {
		var val interface{}
		d := json.NewDecoder(strings.NewReader(content))
		d.UseNumber()
		if err := d.Decode(&val); err == nil {
			switch v := reflect.ValueOf(val); v.Kind() {
			case reflect.Map:
				for k, v := range val.(map[string]interface{}) {
					s.Data[k] = v
				}
				// add to SliceData
			case reflect.Slice:
				s.SendSlice(val.([]interface{}))
				// bounce to rawstring if it is arrayjson, or others
			default:
				s.BounceToRawString = true
			}
		} else if formVal, err := url.ParseQuery(content); err == nil {
			for k, _ := range formVal {
				// make it array if already have key
				if val, ok := s.Data[k]; ok {
					var strArray []string
					strArray = append(strArray, formVal.Get(k))
					// check if previous data is one string or array
					switch oldValue := val.(type) {
					case []string:
						strArray = append(strArray, oldValue...)
					case string:
						strArray = append(strArray, oldValue)
					}
					s.Data[k] = strArray
				} else {
					// make it just string if does not already have same key
					s.Data[k] = formVal.Get(k)
				}
			}
			s.TargetType = "form"
		} else {
			s.BounceToRawString = true
		}
	}
	// Dump all contents to RawString in case in the end user doesn't want json or form.
	s.RawString += content
	return s
}

func changeMapToURLValues(data map[string]interface{}) url.Values {
	var newUrlValues = url.Values{}
	for k, v := range data {
		switch val := v.(type) {
		case string:
			newUrlValues.Add(k, val)
		case []string:
			for _, element := range val {
				newUrlValues.Add(k, element)
			}
			// if a number, change to string
			// json.Number used to protect against a wrong (for GoRequest) default conversion
			// which always converts number to float64.
			// This type is caused by using Decoder.UseNumber()
		case json.Number:
			newUrlValues.Add(k, string(val))
		}
	}
	return newUrlValues
}

// End is the most important function that you need to call when ending the chain. The request won't proceed without calling it.
// End function returns Response which matchs the structure of Response type in Golang's http package (but without Body data). The body data itself returns as a string in a 2nd return value.
// Lastly but worth noticing, error array (NOTE: not just single error value) is returned as a 3rd value and nil otherwise.
//
// For example:
//
//    resp, body, errs := gorequest.New().Get("http://www.google.com").End()
//    if (errs != nil) {
//      fmt.Println(errs)
//    }
//    fmt.Println(resp, body)
//
// Moreover, End function also supports callback which you can put as a parameter.
// This extends the flexibility and makes GoRequest fun and clean! You can use GoRequest in whatever style you love!
//
// For example:
//
//    func printBody(resp gorequest.Response, body string, errs []error){
//      fmt.Println(resp.Status)
//    }
//    gorequest.New().Get("http://www..google.com").End(printBody)
//
func (s *SuperAgent) End(callback ...func(response Response, body string, errs []error)) (Response, string, []error) {
	var bytesCallback []func(response Response, body []byte, errs []error)
	if len(callback) > 0 {
		bytesCallback = []func(response Response, body []byte, errs []error){
			func(response Response, body []byte, errs []error) {
				callback[0](response, string(body), errs)
			},
		}
	}
	resp, body, errs := s.EndBytes(bytesCallback...)
	bodyString := string(body)
	return resp, bodyString, errs
}

// EndBytes should be used when you want the body as bytes. The callbacks work the same way as with `End`, except that a byte array is used instead of a string.
func (s *SuperAgent) EndBytes(callback ...func(response Response, body []byte, errs []error)) (Response, []byte, []error) {
	var (
		req  *http.Request
		err  error
		resp Response
	)
	// check whether there is an error. if yes, return all errors
	if len(s.Errors) != 0 {
		return nil, nil, s.Errors
	}
	// check if there is forced type
	switch s.ForceType {
	case "json", "form", "xml", "text":
		s.TargetType = s.ForceType
		// If forcetype is not set, check whether user set Content-Type header.
		// If yes, also bounce to the correct supported TargetType automatically.
	default:
		for k, v := range Types {
			if s.Header["Content-Type"] == v {
				s.TargetType = k
			}
		}
	}

	// if slice and map get mixed, let's bounce to rawstring
	if len(s.Data) != 0 && len(s.SliceData) != 0 {
		s.BounceToRawString = true
	}

	// Make Request
	req, err = s.MakeRequest()
	if err != nil {
		s.Errors = append(s.Errors, err)
		return nil, nil, s.Errors
	}

	// Set Transport
	if !DisableTransportSwap {
		s.Client.Transport = s.Transport
	}

	// Log details of this request
	if s.Debug {
		dump, err := httputil.DumpRequest(req, true)
		s.logger.SetPrefix("[http] ")
		if err != nil {
			s.logger.Println("Error:", err)
		} else {
			s.logger.Printf("HTTP Request: %s", string(dump))
		}
	}

	// Display CURL command line
	if s.CurlCommand {
		curl, err := http2curl.GetCurlCommand(req)
		s.logger.SetPrefix("[curl] ")
		if err != nil {
			s.logger.Println("Error:", err)
		} else {
			s.logger.Printf("CURL command line: %s", curl)
		}
	}

	// Send request
	resp, err = s.Client.Do(req)
	if err != nil {
		s.Errors = append(s.Errors, err)
		return nil, nil, s.Errors
	}
	defer resp.Body.Close()

	// Log details of this response
	if s.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if nil != err {
			s.logger.Println("Error:", err)
		} else {
			s.logger.Printf("HTTP Response: %s", string(dump))
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	// Reset resp.Body so it can be use again
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	// deep copy response to give it to both return and callback func
	respCallback := *resp
	if len(callback) != 0 {
		callback[0](&respCallback, body, s.Errors)
	}
	return resp, body, nil
}

func (s *SuperAgent) MakeRequest() (*http.Request, error) {
	var (
		req *http.Request
		err error
	)

	switch s.Method {
	case POST, PUT, PATCH:
		if s.TargetType == "json" {
			// If-case to give support to json array. we check if
			// 1) Map only: send it as json map from s.Data
			// 2) Array or Mix of map & array or others: send it as rawstring from s.RawString
			var contentJson []byte
			if s.BounceToRawString {
				contentJson = []byte(s.RawString)
			} else if len(s.Data) != 0 {
				contentJson, _ = json.Marshal(s.Data)
			} else if len(s.SliceData) != 0 {
				contentJson, _ = json.Marshal(s.SliceData)
			}
			contentReader := bytes.NewReader(contentJson)
			req, err = http.NewRequest(s.Method, s.Url, contentReader)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/json")
		} else if s.TargetType == "form" {
			var contentForm []byte
			if s.BounceToRawString || len(s.SliceData) != 0 {
				contentForm = []byte(s.RawString)
			} else {
				formData := changeMapToURLValues(s.Data)
				contentForm = []byte(formData.Encode())
			}
			contentReader := bytes.NewReader(contentForm)
			req, err = http.NewRequest(s.Method, s.Url, contentReader)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else if s.TargetType == "text" {
			req, err = http.NewRequest(s.Method, s.Url, strings.NewReader(s.RawString))
			req.Header.Set("Content-Type", "text/plain")
		} else if s.TargetType == "xml" {
			req, err = http.NewRequest(s.Method, s.Url, strings.NewReader(s.RawString))
			req.Header.Set("Content-Type", "application/xml")
		} else {
			// TODO: if nothing match, let's return warning here
		}
	case "":
		return nil, errors.New("No method specified")
	default:
		req, err = http.NewRequest(s.Method, s.Url, nil)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range s.Header {
		req.Header.Set(k, v)
	}
	// Add all querystring from Query func
	q := req.URL.Query()
	for k, v := range s.QueryData {
		for _, vv := range v {
			q.Add(k, vv)
		}
	}
	req.URL.RawQuery = q.Encode()

	// Add basic auth
	if s.BasicAuth != struct{ Username, Password string }{} {
		req.SetBasicAuth(s.BasicAuth.Username, s.BasicAuth.Password)
	}

	// Add cookies
	for _, cookie := range s.Cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}
