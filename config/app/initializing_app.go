package app

import (
	appContext "j-okx-ai/components/app_context"
	"j-okx-ai/middlewares"
	ginAuth "j-okx-ai/modules/auth/transport/gin"
	ginMock "j-okx-ai/modules/mock/transport/gin"
	ginUser "j-okx-ai/modules/user/transport/gin"

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
		auth := v1.Group("/Auth")
		{
			auth.POST("/Login", ginAuth.Login(appContext.GetMainDBConnection()))
			auth.POST("/Register", ginAuth.Register(appContext.GetMainDBConnection()))
		}

	}

	//API configs
	protected := v1.Group("")
	//protected.Use(middlewares.AuthMiddleware())

	//Protected Auth
	{
		auth := v1.Group("/Auth")
		{
			auth.POST("/RefreshToken", ginAuth.RefreshToken())
		}

	}

	// User API
	{
		{
			protected.POST("/Create/User/", ginUser.CreateUser(appContext.GetMainDBConnection()))
			protected.GET("/Get/User/:id", ginUser.GetUserById(appContext.GetMainDBConnection()))
			protected.GET("/Get/Users", ginUser.GetUsers(appContext.GetMainDBConnection()))
			protected.PUT("/Update/User/:id", ginUser.UpdateUser(appContext.GetMainDBConnection()))
			protected.DELETE("/Delete/User/:id", ginUser.DeleteUser(appContext.GetMainDBConnection()))
			protected.PUT("/Update/Password/User/:id", ginUser.UpdateUserPassword(appContext.GetMainDBConnection()))

		}
	}

	// Mock API
	{
		{
			protected.POST("/Create/Mock", ginMock.CreateMock(appContext.GetMainDBConnection()))
			protected.GET("/Get/Mock/:id", ginMock.GetMockById(appContext.GetMainDBConnection()))
			protected.GET("/Get/Mocks", ginMock.GetMocks(appContext.GetMainDBConnection()))
			protected.PUT("/Update/Mock/:id", ginMock.UpdateMock(appContext.GetMainDBConnection()))
			protected.DELETE("/Delete/Mock/:id", ginMock.DeleteMock(appContext.GetMainDBConnection()))

		}
	}

}
