package router

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/freeFishGo/config"
	"github.com/freeFishGo/httpContext"
	"html/template"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
)

type ControllerRegister struct {
	tree       *tree
	WebConfig  *config.WebConfig
	staticFile map[string]string
}

// 实例化一个mvc注册器

func NewControllerRegister() *ControllerRegister {
	controllerRegister := new(ControllerRegister)
	controllerRegister.tree = newTree()
	controllerRegister.WebConfig = config.NewWebConfig()
	controllerRegister.staticFile = map[string]string{}
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
		return ctx
	}
	ctl := f.ControllerInfo
	action := reflect.New(ctl.ControllerFunc)
	var ic IController = action.Interface().(IController)
	ic.initController(ctx)
	ctx.Request.ParseForm()
	var param interface{}
	ic.setQuery(f.OtherKeyMap)
	if ctl.ControllerActionParameterStruct != nil {
		param = reflect.New(ctl.ControllerActionParameterStruct).Interface()
		data := fromToSimpleMap(ctx.Request.Form, f.OtherKeyMap)
		json.Unmarshal(data, param)
	}
	action.MethodByName(ctl.ControllerAction).Call(getValues(param))
	if !ctx.Response.Started {
		con := ic.getController()
		con.controllerName = ctl.ControllerName
		con.actionName = ctl.ControllerAction
		err := c.tmpHtml(con)
		if err != nil {
			panic(err)
		}
	}
	return ctx
}
func (ctr *ControllerRegister) tmpHtml(c *Controller) error {
	if c.isUseTplPath || c.LayoutPath != "" {
		if c.LayoutPath != "" {
			if b, err := ctr.htmlTpl(c.LayoutPath); err == nil {
				section := map[string]interface{}{}
				if c.tplPath == "" {
					c.tplPath = filepath.Join(c.controllerName, replaceActionName(c.actionName)+".fish")
				}
				if b, err := ctr.htmlTpl(c.tplPath); err == nil {
					section["LayoutContent"] = template.HTML(b)
				} else {
					return errors.New("Controller:" + c.controllerName + "Action:" + c.actionName + "读取模板地址:" + c.tplPath + "时出错:" + err.Error())
				}
				for k, v := range c.LayoutSections {
					if b1, err := ctr.htmlTpl(v); err == nil {
						section[k] = template.HTML(b1)
					} else {
						return errors.New("操作母模板:" + c.LayoutPath + " 读取子模板: " + k + " 子模板地址:" + v + "时出错:" + err.Error())
					}
				}
				var buf bytes.Buffer

				if t, err := template.New(c.tplPath).Delims(ctr.WebConfig.TemplateLeft, ctr.WebConfig.TemplateRight).Parse(string(b)); err == nil {
					if err := t.Execute(&buf, section); err == nil {
						if t, err := template.New(c.tplPath).Delims(ctr.WebConfig.TemplateLeft, ctr.WebConfig.TemplateRight).Parse(buf.String()); err == nil {
							return t.Execute(&c.HttpContext.Response, c.Data)
						} else {
							return errors.New("Controller:" + c.controllerName + "Action:" + c.actionName + "读取模板地址:" + c.tplPath + "时出错:" + err.Error())
						}
					} else {
						return errors.New("格式化模板页时 母版页地址:" + c.LayoutPath + "时出错:" + err.Error())
					}
				} else {
					return errors.New("Controller:" + c.controllerName + "Action:" + c.actionName + "读取模板地址:" + c.tplPath + "时出错:" + err.Error())
				}
			} else {
				return errors.New("Controller:" + c.controllerName + "Action:" + c.actionName + "读取模板地址:" + c.tplPath + "时出错:" + err.Error())
			}
		}
		if c.tplPath == "" {
			c.tplPath = filepath.Join(c.controllerName, replaceActionName(c.actionName)+".fish")
		}
		if b, err := ctr.htmlTpl(c.tplPath); err == nil {
			// 创建一个新的模板，并且载入内容
			if t, err := template.New(c.tplPath).Delims(ctr.WebConfig.TemplateLeft, ctr.WebConfig.TemplateRight).Parse(string(b)); err == nil {
				return t.Execute(&c.HttpContext.Response, c.Data)
			} else {
				return errors.New("Controller:" + c.controllerName + "Action:" + c.actionName + "读取模板地址:" + c.tplPath + "时出错:" + err.Error())
			}
		} else {
			return errors.New("Controller:" + c.controllerName + "Action:" + c.actionName + "读取模板地址:" + c.tplPath + "时出错:" + err.Error())
		}
	}
	return nil
}

func (ctr *ControllerRegister) htmlTpl(path string) (string, error) {
	path = filepath.Join(ctr.WebConfig.ViewsPath, path)
	if v, ok := ctr.staticFile[path]; ok {
		return v, nil
	} else {
		if b, err := ioutil.ReadFile(path); err == nil {
			ctr.staticFile[path] = string(b)
			return ctr.staticFile[path], nil
		} else {
			return "", err
		}
	}
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
			ff := new(freeFishUrl)
			ff.OtherKeyMap = map[string]interface{}{}
			for k, m := range v.patternMap {
				ff.OtherKeyMap[k] = sl[m]
			}
			ff.controllerAction = ff.GetControllerAction(v) + strings.ToLower(string(method))
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
			ff := new(freeFishUrl)
			ff.OtherKeyMap = map[string]interface{}{}
			for k, m := range v.patternMap {
				ff.OtherKeyMap[k] = sl[m]
			}
			ff.controllerAction = ff.GetControllerAction(v) + strings.ToLower(string(method))
			ff.controllerName = ff.GetControllerName(v)
			if v, ok := c.tree.getControllerInfoByControllerNameControllerAction(ff.controllerName, ff.controllerAction); ok {
				ff.ControllerInfo = v
				return ff
			}
		}
	}

	return nil
}
