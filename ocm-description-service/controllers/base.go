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
	"context"
	"flag"
	"icos/server/ocm-description-service/utils/logs"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Server struct {
	Router *mux.Router
}

func (server *Server) Init() {
	server.Router = mux.NewRouter()

	// swagger
	server.Router.PathPrefix("/deploy-manager/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("doc.json"), //The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	server.initializeRoutes()
}

func (server *Server) Run(addr string) {
	logs.Logger.Println("Listening to port " + addr + " ...")
	handler := cors.AllowAll().Handler(server.Router)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	go func() {
		// init server
		if err := http.ListenAndServe(addr, handler); err != nil {
			if err != http.ErrServerClosed {
				logs.Logger.Fatal(err)
			}
		}
	}()

	<-stop

	// after stopping server
	logs.Logger.Println("Closing connections ...")

	var shutdownTimeout = flag.Duration("shutdown-timeout", 10*time.Second, "shutdown timeout (5s,5m,5h) before connections are cancelled")
	_, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()
}
