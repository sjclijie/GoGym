package GoGym

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"github.com/golang/glog"
)

// HTTP Methods
const (
	GETMethod     = "GET"
	POSTMethod    = "POST"
	PUTMethod     = "PUT"
	PATCHMethod   = "PATCH"
	DELETEMethod  = "DELETE"
	OPTIONSMethod = "OPTIONS"
)

//MIME Types
const (
	MIME_APP_JSON = "application/json"
	//MIME_APP_JSON_CHARSETUTF8 = MIME_APP_JSON + ";" + "charset=UTF-8"
	//MIME_APP_FORM = "application/x-www-form-urlencoded"
	//MIME_APP_PB   = "application/protobuf"
)

//HTTP Codes
const (
	HTTPMethodNotAllowed = http.StatusMethodNotAllowed
	//HTTPNotFound = http.StatusNotFound
	//HTTPUnauthorized = http.StatusUnauthorized
	//HTTPTooManyRequests = http.StatusTooManyRequests
	//HTTPBadGateway = http.StatusBadGateway
	//HTTPBadRequest = http.StatusBadRequest
)

// APIService for now is the struct for containing controllerRegistry and registeredPathAndController,
// and it is the core service provider
type APIService struct {
	// controllerRegistry is where all registered controllers exist
	controllerRegistry map[string]interface{}
	//registeredPathAndController is a mapping of paths and controllers
	registeredPathAndController map[string]map[string]map[string]string // TODO：optimize the data structure of registeredPathAndController

	requestForm                 map[string]url.Values
}

func (api *APIService) Get(path, controllerWithActionString string) {
	mapping := api.mappingRequestMethodWithControllerAndActions(GETMethod, path, controllerWithActionString)
	api.registeredPathAndController[path] = mapping
}

func (api *APIService) Post(path, controllerWithActionString string) {
	mapping := api.mappingRequestMethodWithControllerAndActions(POSTMethod, path, controllerWithActionString)
	api.registeredPathAndController[path] = mapping
}

func (api *APIService) Put(path, controllerWithActionString string) {
	mapping := api.mappingRequestMethodWithControllerAndActions(PUTMethod, path, controllerWithActionString)
	api.registeredPathAndController[path] = mapping
}

func (api *APIService) Patch(path, controllerWithActionString string) {
	mapping := api.mappingRequestMethodWithControllerAndActions(PATCHMethod, path, controllerWithActionString)
	api.registeredPathAndController[path] = mapping
}

func (api *APIService) Options(path, controllerWithActionString string) {
	mapping := api.mappingRequestMethodWithControllerAndActions(OPTIONSMethod, path, controllerWithActionString)
	api.registeredPathAndController[path] = mapping
}

func (api *APIService) Delete(path, controllerWithActionString string) {
	mapping := api.mappingRequestMethodWithControllerAndActions(DELETEMethod, path, controllerWithActionString)
	api.registeredPathAndController[path] = mapping
}

// mappingRequestMethodWithControllerAndActions is a function for mapping request method with controllers
// which containing actions
func (api *APIService) mappingRequestMethodWithControllerAndActions(requestMethod, path, controllerWithActionString string) map[string]map[string]string {
	mappingResult := make(map[string]map[string]string)
	if length := len(api.registeredPathAndController[path]); length > 0 {
		mappingResult = api.registeredPathAndController[path]
	}
	controllerAndActionSlice := strings.Split(controllerWithActionString, "@")
	controller := controllerAndActionSlice[0]
	action := controllerAndActionSlice[1]
	controllerAndActionMap := map[string]string{controller: action}
	mappingResult[requestMethod] = controllerAndActionMap
	return mappingResult
}

// HandleRequest is a function to handle http request
func (api *APIService) HandleRequest(controllers map[string]map[string]string) http.HandlerFunc {
	return func(rw http.ResponseWriter, request *http.Request) {
		request.ParseForm()
		method := request.Method
		api.requestForm["query"] = request.Form
		api.requestForm["form"] = request.PostForm
		macthedControllers, ok := controllers[method]
		if !ok {
			rw.WriteHeader(HTTPMethodNotAllowed)
		}
		for k, v := range macthedControllers {
			controllerKey := "*" + k
			controller := api.controllerRegistry[controllerKey]
			in := make([]reflect.Value, 2)
			in[0] = reflect.ValueOf(api.requestForm)
			in[1] = reflect.ValueOf(request.Header)
			returnValues := reflect.ValueOf(controller).MethodByName(v).Call(in)
			statusCode := returnValues[0].Interface()
			intStatusCode := statusCode.(int)
			response := returnValues[1].Interface()
			api.JSONResponse(rw, intStatusCode, response)
		}
	}
}

// RegisterHandleFunc is a function registers a handle function to handle request from path
func (api *APIService) RegisterHandleFunc() {
	for k, v := range api.registeredPathAndController {
		path := k
		if !strings.HasPrefix(k, "/") {
			path = fmt.Sprintf("/%v", k)
		}
		http.HandleFunc(path, api.HandleRequest(v))
	}
}

// RegisterControllers is a function registers a struct of controllers into controllerRegistry
func (api *APIService) RegisterControllers(controllers []interface{}) {
	for _, v := range controllers {
		api.RegisterController(v)
	}
}

// RegisterControllers is a function registers a controller into controllerRegistry
func (api *APIService) RegisterController(controller interface{}) {
	controllerType := getType(controller)
	api.controllerRegistry[controllerType] = controller
}

// getType is a function gets the type of value
func getType(value interface{}) string {
	if t := reflect.TypeOf(value); t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	} else {
		return t.Name()
	}
}

// Serve is a function
func (api *APIService) Serve(port int) {
	api.RegisterHandleFunc()
	fullPort := fmt.Sprintf(":%d", port)
	http.ListenAndServe(fullPort, nil)
}

// JSONResponse is a function return json response
func (api *APIService) JSONResponse(rw http.ResponseWriter, statusCode int, response interface{}) {
	// TODO: accept headers
	rw.Header().Set("Content-Type", MIME_APP_JSON)
	rw.WriteHeader(statusCode)
	rsp, err := json.Marshal(response)
	if err != nil {
		// TODO: logging error
		glog.Error(fmt.Sprintf("JSON err: %s", err))
		//fmt.Println("JSON err:", err)
	}
	rw.Write(rsp)
}

// Prepare is a fucntion prepare the service and return prepared service to the user
func Prepare() *APIService {
	var apiService = new(APIService)
	apiService.controllerRegistry = make(map[string]interface{})
	apiService.registeredPathAndController = make(map[string]map[string]map[string]string)
	apiService.requestForm = make(map[string]url.Values)
	return apiService
}
