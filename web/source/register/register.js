/*
	GoToSocial
	Copyright (C) GoToSocial Authors admin@gotosocial.org
	SPDX-License-Identifier: AGPL-3.0-or-later

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

"use strict";

const React = require("react");

const query = require("../settings/lib/query");

const {
	useTextInput,
	useBoolInput,
} = require("../settings/lib/form");

const useFormSubmit = require("../settings/lib/form/submit");


const {
	Select,
	TextInput,
	Checkbox
} = require("../settings/components/form/inputs");

const FormWithData = require("../settings/lib/form/form-with-data");
const Languages = require("../settings/components/languages");
const MutationButton = require("../settings/components/form/mutation-button");

function UserRegisterForm({ data }) {
  const form = {
    username:  useTextInput("source[username]",  { source: data, defaultValue: "nothing" }),
    email:     useTextInput("source[email]",     { source: data, defaultValue: "nothing" }),
    password:  useTextInput("source[password]",  { source: data, defaultValue: "nothing" }),
    reason:    useTextInput("source[reason]",    { source: data, defaultValue: "nothing" }),
    agreement: useBoolInput("source[agreement]", { source: data }),
    locale:    useTextInput("source[locale]",    { source: data, valueSelector: (s) => s.source.language?.toUpperCase() ?? "EN" }),
  };



  const [submitForm, result] = useFormSubmit(form, query.useAuthorizeFlowMutation());

  return (
   <form className="register" onsubmit={submitForm}>
      <div> test </div>
  </form>
  );
}

module.exports = { UserRegisterForm };
