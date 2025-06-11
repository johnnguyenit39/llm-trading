package app

import (
	appContext "j-ai-trade/components/app_context"
	"j-ai-trade/middlewares"
	ginAuth "j-ai-trade/modules/auth/transport/gin"
	ginMock "j-ai-trade/modules/okx/transport/gin"
	ginUser "j-ai-trade/modules/user/transport/gin"

	"github.com/gin-contrib/cors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitializeApp(appContext appContext.AppContext) {
	router := appContext.GetGinApp()

	// Add swagger
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

	// Okx API
	{
		{
			protected.POST("/create/mock", ginMock.CreateMock(appContext.GetMainDBConnection()))
			protected.GET("/get/okx/:id", ginMock.GetMockById(appContext.GetMainDBConnection()))
			protected.GET("/get/okx-info", ginMock.GetOkxInfo(appContext.GetMainDBConnection()))
			protected.GET("/get/mocks", ginMock.GetMocks(appContext.GetMainDBConnection()))
			protected.PUT("/update/okx/:id", ginMock.UpdateMock(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/okx/:id", ginMock.DeleteMock(appContext.GetMainDBConnection()))
			protected.POST("/create/order", ginMock.CreateOrder(appContext.GetMainDBConnection()))
			protected.POST("/cancel/order", ginMock.CancelOrder(appContext.GetMainDBConnection()))
		}
	}

}
