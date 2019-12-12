package router

import (
	"encoding/json"
	"fmt"
	"freeFishGo/httpContext"
	"html/template"
	"io/ioutil"
	"log"
	"net/url"
	"reflect"
	"strings"
)

type ControllerRegister struct {
	tree *tree
}

func NewControllerRegister() *ControllerRegister {
	controllerRegister := new(ControllerRegister)
	controllerRegister.tree = newTree()
	return controllerRegister
}

func (cr *ControllerRegister) AddHandlers(ctl IController) {
	ctl.setSonController(ctl)
	cr.tree = ctl.getControllerInfo(cr.tree)
}

// 主路由节点注册，必须含有{Controller}和{Action}变量
func (cr *ControllerRegister) AddMainRouter(ctlList ...*ControllerActionInfo) {
	cr.tree.MainRouterList = cr.tree.MainRouterList.AddControllerModelList(ctlList...)
}

// 如果主路由为空注册一个默认主路由
func (cr *ControllerRegister) MainRouterNil() {
	if cr.tree.MainRouterList == nil || len(cr.tree.MainRouterList) == 0 {
		cr.tree.MainRouterList = cr.tree.MainRouterList.AddControllerModelList(&ControllerActionInfo{RouterPattern: "/{ Controller}/{Action}"})
	}
}

// http服务逻辑处理程序
//func (c *ControllerRegister) Middleware(ctx *httpContext.HttpContext) {
//	c.AnalysisRequest(ctx)
//}
func (c *ControllerRegister) AnalysisRequest(ctx *httpContext.HttpContext) *httpContext.HttpContext {
	u, _ := url.Parse(ctx.Request.RequestURI)
	f := c.analysisUrlToGetAction(u, httpContext.HttpMethod(ctx.Request.Method))
	if f == nil {
		ctx.Response.WriteHeader(404)
		ctx.Response.Write([]byte("404错误"))
		return ctx
	}
	ctl, ok := c.tree.getControllerInfoByControllerNameControllerAction(f.controllerName, f.controllerAction)
	if ok {
		action := reflect.New(ctl.ControllerFunc)
		var ic IController = action.Interface().(IController)
		ic.initController(ctx)
		ctx.Request.ParseForm()
		var param interface{}
		if ctl.ControllerActionParameterStruct != nil {
			param = reflect.New(ctl.ControllerActionParameterStruct).Interface()
			data := fromToSimpleMap(ctx.Request.Form, f.OtherKeyMap)
			json.Unmarshal(data, param)
		}
		log.Println(fmt.Sprintf("数据：%+v", param))
		action.MethodByName(ctl.ControllerAction).Call(getValues(param))
		if !ctx.Response.Started {
			c := ic.getController()
			tmpHtml(c)
			log.Println(fmt.Sprintf("数据：%+v", ic.getController()))
		}
	} else {
		ctx.Response.WriteHeader(404)
		ctx.Response.Write([]byte("404错误"))
	}
	return ctx
}

func tmpHtml(c *Controller) (err error) {
	if b, err := ioutil.ReadFile(c.TplPath); err == nil {
		// 创建一个新的模板，并且载入内容
		log.Println(string(b))
		if t, err := template.New("webpage").Parse(string(b)); err == nil {
			return t.Execute(c.HttpContext.Response, c.Data)
		}
	} else {
		panic(err.Error())
	}
	return
}

//根据参数获取对应的Values
func getValues(param ...interface{}) []reflect.Value {
	vals := make([]reflect.Value, 0, len(param))
	for i := range param {
		vals = append(vals, reflect.ValueOf(param[i]))
	}
	return vals
}

func fromToSimpleMap(v url.Values, addKeyVal map[string]interface{}) []byte {
	dic := map[string]interface{}{}
	for k, val := range v {
		if len(val) == 1 {
			dic[k] = val[0]
		} else {
			dic[k] = val
		}
	}
	for k, val := range addKeyVal {
		dic[k] = val
	}
	data, err := json.Marshal(dic)
	if err != nil {
		panic(err.Error())
	}
	return data
}

// 根据url对象分析出控制处理器名称，并把其他规则数据提取出来
func (c *ControllerRegister) analysisUrlToGetAction(u *url.URL, method httpContext.HttpMethod) *freeFishUrl {
	path := strings.ToLower(u.Path)
	for _, v := range c.tree.MainRouterList {
		sl := v.patternRe.FindStringSubmatch(path)
		if len(sl) != 0 {
			if _, ok := v.allowMethod[method]; !ok {
				continue
			}
			ff := new(freeFishUrl)
			ff.OtherKeyMap = map[string]interface{}{}
			for k, m := range v.patternMap {
				ff.OtherKeyMap[k] = sl[m]
			}
			ff.controllerAction = ff.GetControllerAction(v)
			ff.controllerName = ff.GetControllerName(v)
			if v, ok := c.tree.getControllerInfoByControllerNameControllerAction(ff.controllerName, ff.controllerAction); ok {
				ff.ControllerInfo = v
				return ff
			}
		}
	}

	for _, v := range c.tree.ControllerModelList {
		sl := v.patternRe.FindStringSubmatch(path)
		if len(sl) != 0 {
			if _, ok := v.allowMethod[method]; !ok {
				continue
			}
			ff := new(freeFishUrl)
			ff.OtherKeyMap = map[string]interface{}{}
			for k, m := range v.patternMap {
				ff.OtherKeyMap[k] = sl[m]
			}
			ff.controllerAction = ff.GetControllerAction(v)
			ff.controllerName = ff.GetControllerName(v)
			if v, ok := c.tree.getControllerInfoByControllerNameControllerAction(ff.controllerName, ff.controllerAction); ok {
				ff.ControllerInfo = v
				return ff
			}
		}
	}

	return nil
}
