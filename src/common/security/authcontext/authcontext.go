// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authcontext

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/vmware/harbor/src/ui/config"
)

const (
	// AuthTokenHeader is the key of auth token header
	AuthTokenHeader  = "x-xenon-auth-token"
	sysAdminRole     = "CLOUD_ADMIN"
	projectAdminRole = "DEVOPS_ADMIN"
	developerRole    = "DEVELOPER"
	guestRole        = "GUEST"
)

var client = &http.Client{
	Transport: &http.Transport{},
}

// AuthContext ...
type AuthContext struct {
	PrincipalID string              `json:"principalId"`
	Name        string              `json:"name"`
	Roles       []string            `json:"projects"`
	Projects    map[string][]string `json:"roles"`
}

// GetUsername ...
func (a *AuthContext) GetUsername() string {
	return a.PrincipalID
}

// IsSysAdmin ...
func (a *AuthContext) IsSysAdmin() bool {
	isSysAdmin := false
	for _, role := range a.Roles {
		// TODO update the value of role when admiral API is ready
		if role == sysAdminRole {
			isSysAdmin = true
			break
		}
	}
	return isSysAdmin
}

// HasReadPerm ...
func (a *AuthContext) HasReadPerm(project string) bool {
	_, exist := a.Projects[project]
	return exist
}

// HasWritePerm ...
func (a *AuthContext) HasWritePerm(project string) bool {
	roles, _ := a.Projects[project]
	for _, role := range roles {
		if role == projectAdminRole || role == developerRole {
			return true
		}
	}
	return false
}

// HasAllPerm ...
func (a *AuthContext) HasAllPerm(project string) bool {
	roles, _ := a.Projects[project]
	for _, role := range roles {
		if role == projectAdminRole {
			return true
		}
	}
	return false
}

// GetByToken ...
func GetByToken(token string) (*AuthContext, error) {
	req, err := http.NewRequest(http.MethodGet, buildCtxURL(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add(AuthTokenHeader, token)

	code, _, data, err := send(req)

	if code != http.StatusOK {
		return nil, fmt.Errorf("failed to get auth context by token: %d %s",
			code, string(data))
	}

	ctx := &AuthContext{
		Projects: make(map[string][]string),
	}
	if err = json.Unmarshal(data, ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

// Login ...
func Login(username, password string) (string, *AuthContext, error) {
	data, err := json.Marshal(&struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest(http.MethodPost, buildLoginURL(), bytes.NewBuffer(data))
	if err != nil {
		return "", nil, err
	}

	code, header, data, err := send(req)
	if code != http.StatusOK {
		return "", nil, fmt.Errorf("failed to login with user %s: %d %s", username,
			code, string(data))
	}

	ctx := &AuthContext{
		Projects: make(map[string][]string),
	}
	if err = json.Unmarshal(data, ctx); err != nil {
		return "", nil, err
	}

	return header.Get(AuthTokenHeader), ctx, nil
}

func send(req *http.Request) (int, http.Header, []byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, err
	}
	return resp.StatusCode, resp.Header, data, nil
}

func buildCtxURL() string {
	return strings.TrimRight(config.AdmiralEndpoint(), "/") + "/sso/auth-context"
}

func buildLoginURL() string {
	return strings.TrimRight(config.AdmiralEndpoint(), "/") + "/sso/login"
}
