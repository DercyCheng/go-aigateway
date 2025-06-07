package router

import (
	"go-aigateway/internal/config"
	"go-aigateway/internal/handlers"
	"go-aigateway/internal/localmodel"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SetupLocalModelRoutes sets up routes for the local model
func SetupLocalModelRoutes(r *gin.Engine, manager *localmodel.Manager, cfg *config.Config) {
	if !cfg.LocalModel.Enabled {
		logrus.Info("Local model is disabled")
		return
	}

	// Create model manager
	modelManager := localmodel.NewModelManager(cfg.LocalModel.ModelPath, cfg.LocalModel.PythonPath)

	// Create handlers
	handler := handlers.NewLocalModelHandler(manager, &cfg.LocalModel)
	managerHandler := handlers.NewLocalModelManagerHandler(manager, modelManager, &cfg.LocalModel)

	// Register routes
	handlers.RegisterLocalModelRoutes(r, handler)
	handlers.RegisterLocalModelManagerRoutes(r, managerHandler)
}
