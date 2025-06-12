package app

import (
	appContext "j-ai-trade/components/app_context"
	"j-ai-trade/middlewares"
	ginApiKey "j-ai-trade/modules/api_key/transport/gin"
	ginAuth "j-ai-trade/modules/auth/transport/gin"
	ginOkx "j-ai-trade/modules/okx/transport/gin"
	ginPermission "j-ai-trade/modules/permission/transport/gin"
	ginSubscription "j-ai-trade/modules/subscription/transport/gin"
	ginUser "j-ai-trade/modules/user/transport/gin"

	"github.com/gin-contrib/cors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitializeApp(appContext appContext.AppContext) {
	router := appContext.GetGinApp()

	// Add swagger
	swaggerGroup := router.Group("/docs")
	// Add basic auth middleware for Swagger UI
	swaggerGroup.Use(BasicAuthMiddleware())

	// Add swagger
	swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Use CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Apply middleware
	router.Use(middlewares.PanicRecoveryMiddleware())
	router.Use(middlewares.RequestIDMiddleware())
	router.Use(middlewares.LoggerMiddleware())

	v1 := router.Group("/api/v1")

	// Authentication routes (no authentication required)
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", ginAuth.Login(appContext.GetMainDBConnection()))
			auth.POST("/register", ginAuth.Register(appContext.GetMainDBConnection()))
		}

	}

	//API configs
	protected := v1.Group("")
	//protected.Use(middlewares.AuthMiddleware())

	//Protected Auth
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/refresh-token", ginAuth.RefreshToken())
		}

	}

	// User API
	{
		{
			protected.POST("/create/user/", ginUser.CreateUser(appContext.GetMainDBConnection()))
			protected.GET("/get/user/:id", ginUser.GetUserById(appContext.GetMainDBConnection()))
			protected.GET("/get/users", ginUser.GetUsers(appContext.GetMainDBConnection()))
			protected.PUT("/update/user/:id", ginUser.UpdateUser(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/user/:id", ginUser.DeleteUser(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/user/:id", ginUser.UpdateUserPassword(appContext.GetMainDBConnection()))

		}
	}

	// Subscription API
	{
		{
			protected.POST("/create/user/", ginSubscription.CreateSubscription(appContext.GetMainDBConnection()))
			protected.GET("/get/user/:id", ginSubscription.GetSubscriptionById(appContext.GetMainDBConnection()))
			protected.GET("/get/users", ginSubscription.GetSubscriptions(appContext.GetMainDBConnection()))
			protected.PUT("/update/user/:id", ginSubscription.UpdateSubscription(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/user/:id", ginSubscription.DeleteSubscription(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/user/:id", ginSubscription.UpdateSubscription(appContext.GetMainDBConnection()))

		}
	}

	// ApiKey API
	{
		{
			protected.POST("/create/user/", ginApiKey.CreateApiKey(appContext.GetMainDBConnection()))
			protected.GET("/get/user/:id", ginApiKey.GetApiKeyById(appContext.GetMainDBConnection()))
			protected.GET("/get/users", ginApiKey.GetApiKeys(appContext.GetMainDBConnection()))
			protected.PUT("/update/user/:id", ginApiKey.UpdateApiKey(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/user/:id", ginApiKey.DeleteApiKey(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/user/:id", ginApiKey.UpdateApiKey(appContext.GetMainDBConnection()))

		}
	}

	// Permission API
	{
		{
			protected.POST("/create/permission/", ginPermission.CreatePermission(appContext.GetMainDBConnection()))
			protected.GET("/get/permission/:id", ginPermission.GetPermissionById(appContext.GetMainDBConnection()))
			protected.GET("/get/permissions", ginPermission.GetPermissions(appContext.GetMainDBConnection()))
			protected.PUT("/update/permission/:id", ginPermission.UpdatePermission(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/permission/:id", ginPermission.DeletePermission(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/permission/:id", ginPermission.UpdatePermission(appContext.GetMainDBConnection()))

		}
	}

	// Okx API
	{
		{
			protected.POST("/create/mock", ginOkx.CreateSubscription(appContext.GetMainDBConnection()))
			protected.GET("/get/okx/:id", ginOkx.GetSubscriptionById(appContext.GetMainDBConnection()))
			protected.GET("/get/okx-info", ginOkx.GetOkxInfo(appContext.GetMainDBConnection()))
			protected.GET("/get/mocks", ginOkx.GetSubscriptions(appContext.GetMainDBConnection()))
			protected.PUT("/update/okx/:id", ginOkx.UpdateSubscription(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/okx/:id", ginOkx.DeleteSubscription(appContext.GetMainDBConnection()))
			protected.POST("/create/order", ginOkx.CreateOrder(appContext.GetMainDBConnection()))
			protected.POST("/cancel/order", ginOkx.CancelOrder(appContext.GetMainDBConnection()))
		}
	}

}
