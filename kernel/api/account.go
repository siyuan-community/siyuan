// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package api

import (
	"net/http"

	"github.com/88250/gulu"
	"github.com/gin-gonic/gin"
	"github.com/siyuan-community/siyuan/kernel/model"
	"github.com/siyuan-community/siyuan/kernel/util"
)

func startFreeTrial(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	err := model.StartFreeTrial()
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}
}

func useActivationcode(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	code := arg["data"].(string)
	err := model.UseActivationcode(code)
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}
}

func checkActivationcode(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	code := arg["data"].(string)
	ret.Code, ret.Msg = model.CheckActivationcode(code)
}

func deactivateUser(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	err := model.DeactivateUser()
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}
}

func login(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	arg, ok := util.JsonArg(c, ret)
	if !ok {
		c.JSON(http.StatusOK, ret)
		return
	}

	name := arg["userName"].(string)
	password := arg["userPassword"].(string)
	captcha := arg["captcha"].(string)
	cloudRegion := int(arg["cloudRegion"].(float64))
	ret = model.Login(name, password, captcha, cloudRegion)
	c.JSON(http.StatusOK, ret)
}
