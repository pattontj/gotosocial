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
const ReactDom = require("react-dom/client");
const { Provider } = require("react-redux");
const { PersistGate } = require("redux-persist/integration/react");

const { UserRegisterForm } = require("./register.js");

// const { store, persistor } = require("./redux");
// const { createNavigation, Menu, Item } = require("./lib/navigation");

// const AuthorizationGate = require("./components/authorization");
// const Loading = require("./components/loading");
// const UserLogoutCard = require("./components/user-logout-card");
// const { RoleContext } = require("./lib/navigation/util");

// require("./style.css");

// const { Sidebar, ViewRouter } = createNavigation("/register", [


		// Item("Settings", { icon: "fa-cogs" }, require("./user/settings")),
  
	// Menu("User", [
	// 	Item("Profile", { icon: "fa-user" }, require("./user/profile")),
	// 	Item("Settings", { icon: "fa-cogs" }, require("./user/settings")),
	// ]),

function App() {
		return (
	  <div> 
      <UserRegisterForm />
    </div>	
  );
}

function Main() {
	return (
		// <Provider store={store}>
		// 	<PersistGate loading={<section><Loading /></section>} persistor={persistor}>
		// 		<AuthorizationGate App={App} />
		// 	</PersistGate>
		// </Provider>
  <App />
	);
}

const root = ReactDom.createRoot(document.getElementById("root"));
root.render(<React.StrictMode><Main /></React.StrictMode>);