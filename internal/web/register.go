// GoToSocial
// Copyright (C) GoToSocial Authors admin@gotosocial.org
// SPDX-License-Identifier: AGPL-3.0-or-later
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
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package web

import (
	// "context"
	// "errors"
	// "fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	// accountcreate "github.com/superseriousbusiness/gotosocial/internal/api/client/accounts"
	// "github.com/superseriousbusiness/gotosocial/cmd/gotosocial/action/admin/account"
	// apimodel "github.com/superseriousbusiness/gotosocial/internal/api/client/model"
	apiutil "github.com/superseriousbusiness/gotosocial/internal/api/util"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	// "github.com/superseriousbusiness/gotosocial/internal/oauth"
	// "golang.org/x/crypto/bcrypt"
)

// Wrap form values for post
type registration struct {
	Username  string `form:"username"`
	Email     string `form:"email"`
	Password  string `form:"password"`
	Reason    string `form:"reason"`
	Agreement bool   `form:"agreement"`
	Locale    string `form:"locale"`
}

// SignInGETHandler should be served at https://example.org/auth/sign_in.
// The idea is to present a sign in page to the user, where they can enter their username and password.
// The form will then POST to the sign in page, which will be handled by SignInPOSTHandler.
// If an idp provider is set, then the user will be redirected to that to do their sign in.
func (m *Module) RegisterGETHandler(c *gin.Context) {
	if _, err := apiutil.NegotiateAccept(c, apiutil.HTMLAcceptHeaders...); err != nil {
		apiutil.ErrorHandler(c, gtserror.NewErrorNotAcceptable(err, err.Error()), m.processor.InstanceGetV1)
		return
	}

	if !config.GetOIDCEnabled() {
		instance, errWithCode := m.processor.InstanceGetV1(c.Request.Context())
		if errWithCode != nil {
			apiutil.ErrorHandler(c, errWithCode, m.processor.InstanceGetV1)
			return
		}

		// no idp provider, use our own funky little sign in page
		c.HTML(http.StatusOK, "register.tmpl", gin.H{
			"instance": instance,
		})
		return
	}

	c.Redirect(http.StatusOK, "404.tmpl")
}

// SignInPOSTHandler should be served at https://example.org/auth/sign_in.
// The idea is to present a sign in page to the user, where they can enter their username and password.
// The handler will then redirect to the auth handler served at /auth
// func (m *Module) RegisterPOSTHandler(c *gin.Context) {

// 	form := &apimodel.AccountCreateRequest{}

// 	if err := c.ShouldBind(form); err != nil {
// 		// TODO: handle binding error here!
// 		panic(err)
// 	}

// 	c.Redirect(http.StatusFound, "404.tmpl")
// }
