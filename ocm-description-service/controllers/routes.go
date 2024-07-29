/*
  OCM-DESCRIPTION-SERVICE
  Copyright © 2022-2024 EVIDEN

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

  This work has received funding from the European Union's HORIZON research
  and innovation programme under grant agreement No. 101070177.
*/

package controllers

import (
	m "icos/server/ocm-description-service/middlewares"
)

func (s *Server) initializeRoutes() {

	// Home Route
	s.Router.HandleFunc("/deploy-manager", m.SetMiddlewareLog(m.SetMiddlewareJSON(s.Home))).Methods("GET")
	//healthcheck
	s.Router.HandleFunc("/deploy-manager/healthz", s.HealthCheck).Methods("GET")
	//ocm-descriptor routes
	s.Router.HandleFunc("/deploy-manager/execute", m.SetMiddlewareLog(s.PullJobs)).Methods("GET")
	// get resource (status)
	s.Router.HandleFunc("/deploy-manager/resource", m.SetMiddlewareLog(m.SetMiddlewareJSON(s.GetResourceStatus))).Methods("GET")
	// trigger resource syncup
	s.Router.HandleFunc("/deploy-manager/resource/sync", m.SetMiddlewareLog(m.SetMiddlewareJSON(s.StartSyncUp))).Methods("GET")
}
