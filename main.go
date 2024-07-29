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

package main

import (
	_ "icos/server/docs"
	ocm_description_service "icos/server/ocm-description-service"
)

//	@title			Swagger Deployment Manager API
//	@version		1.0
//	@description	ICOS Deployment Manager Microservice.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8083
//	@BasePath	/

//	@securityDefinitions.basic	OAuth 2.0

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func main() {

	ocm_description_service.Run()

}
