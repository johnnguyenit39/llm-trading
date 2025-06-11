package appcontext

import (
	"j-ai-trade/config/pubsub"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AppContext interface {
	GetMainDBConnection() *gorm.DB
	GetGinApp() *gin.Engine
	GetClient() *redis.Client
	GetPubSub() *pubsub.PubSub
}

type appContext struct {
	db     *gorm.DB
	app    *gin.Engine
	cache  *redis.Client
	pubSub *pubsub.PubSub
}

func NewAppContext(db *gorm.DB, cache *redis.Client, pusSub *pubsub.PubSub, app *gin.Engine) *appContext {
	return &appContext{db: db, cache: cache, app: app, pubSub: pusSub}
}

func (context *appContext) GetMainDBConnection() *gorm.DB {
	return context.db
}

func (context *appContext) GetGinApp() *gin.Engine {
	return context.app
}

func (context *appContext) GetClient() *redis.Client {
	return context.cache
}

func (context *appContext) GetPubSub() *pubsub.PubSub {
	return context.pubSub
}
