package component

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type AppContext interface {
	GetMainDBConnection() *mongo.Database
	GetGinApp() *gin.Engine
}

type appContext struct {
	db  *mongo.Database
	app *gin.Engine
}

func NewAppContext(db *mongo.Database, app *gin.Engine) *appContext {
	return &appContext{db: db, app: app}
}

func (context *appContext) GetMainDBConnection() *mongo.Database {
	return context.db
}

func (context *appContext) GetGinApp() *gin.Engine {
	return context.app
}
