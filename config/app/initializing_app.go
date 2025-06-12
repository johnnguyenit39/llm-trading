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

	ginAiExpert "j-ai-trade/modules/ai_expert/transport/gin"
	ginJbot "j-ai-trade/modules/j_bot/transport/gin"
	ginOrder "j-ai-trade/modules/order/transport/gin"
	ginOtp "j-ai-trade/modules/otp/transport/gin"
	ginSignal "j-ai-trade/modules/signal/transport/gin"

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
			protected.POST("/create/subscription/", ginSubscription.CreateSubscription(appContext.GetMainDBConnection()))
			protected.GET("/get/subscription/:id", ginSubscription.GetSubscriptionById(appContext.GetMainDBConnection()))
			protected.GET("/get/subscription/list", ginSubscription.GetSubscriptions(appContext.GetMainDBConnection()))
			protected.PUT("/update/subscription/:id", ginSubscription.UpdateSubscription(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/subscription/:id", ginSubscription.DeleteSubscription(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/subscription/:id", ginSubscription.UpdateSubscription(appContext.GetMainDBConnection()))

		}
	}

	// ApiKey API
	{
		{
			protected.POST("/create/api-key/", ginApiKey.CreateApiKey(appContext.GetMainDBConnection()))
			protected.GET("/get/api-key/:id", ginApiKey.GetApiKeyById(appContext.GetMainDBConnection()))
			protected.GET("/get/api-key/list", ginApiKey.GetApiKeys(appContext.GetMainDBConnection()))
			protected.PUT("/update/api-key/:id", ginApiKey.UpdateApiKey(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/api-key/:id", ginApiKey.DeleteApiKey(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/api-key/:id", ginApiKey.UpdateApiKey(appContext.GetMainDBConnection()))

		}
	}

	// Permission API
	{
		{
			protected.POST("/create/permission/", ginPermission.CreatePermission(appContext.GetMainDBConnection()))
			protected.GET("/get/permission/:id", ginPermission.GetPermissionById(appContext.GetMainDBConnection()))
			protected.GET("/get/permission/list", ginPermission.GetPermissions(appContext.GetMainDBConnection()))
			protected.PUT("/update/permission/:id", ginPermission.UpdatePermission(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/permission/:id", ginPermission.DeletePermission(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/permission/:id", ginPermission.UpdatePermission(appContext.GetMainDBConnection()))

		}
	}

	// AiExpert API
	{
		{
			protected.POST("/create/ai-expert/", ginAiExpert.CreateAiExpert(appContext.GetMainDBConnection()))
			protected.GET("/get/ai-expert/:id", ginAiExpert.GetAiExpertById(appContext.GetMainDBConnection()))
			protected.GET("/get/ai-expert/list", ginAiExpert.GetAiExperts(appContext.GetMainDBConnection()))
			protected.PUT("/update/ai-expert/:id", ginAiExpert.UpdateAiExpert(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/ai-expert/:id", ginAiExpert.DeleteAiExpert(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/ai-expert/:id", ginAiExpert.UpdateAiExpert(appContext.GetMainDBConnection()))

		}
	}

	// Order API
	{
		{
			protected.POST("/create/order/", ginOrder.CreateOrder(appContext.GetMainDBConnection()))
			protected.GET("/get/order/:id", ginOrder.GetOrderById(appContext.GetMainDBConnection()))
			protected.GET("/get/order/list", ginOrder.GetOrders(appContext.GetMainDBConnection()))
			protected.PUT("/update/order/:id", ginOrder.UpdateOrder(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/order/:id", ginOrder.DeleteOrder(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/ordery/:id", ginOrder.UpdateOrder(appContext.GetMainDBConnection()))

		}
	}

	// Signal API
	{
		{
			protected.POST("/create/signal/", ginSignal.CreateSignal(appContext.GetMainDBConnection()))
			protected.GET("/get/signal/:id", ginSignal.GetSignalById(appContext.GetMainDBConnection()))
			protected.GET("/get/signal/list", ginSignal.GetSignals(appContext.GetMainDBConnection()))
			protected.PUT("/update/signal/:id", ginSignal.UpdateSignal(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/signal/:id", ginSignal.DeleteSignal(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/signal/:id", ginSignal.UpdateSignal(appContext.GetMainDBConnection()))

		}
	}

	// Otp API
	{
		{
			protected.POST("/create/otp/", ginOtp.CreateOtp(appContext.GetMainDBConnection()))
			protected.GET("/get/otp/:id", ginOtp.GetOtpById(appContext.GetMainDBConnection()))
			protected.GET("/get/otp/list", ginOtp.GetOtps(appContext.GetMainDBConnection()))
			protected.PUT("/update/otp/:id", ginOtp.UpdateOtp(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/otp/:id", ginOtp.DeleteOtp(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/otp/:id", ginOtp.UpdateOtp(appContext.GetMainDBConnection()))

		}
	}

	// JBot API
	{
		{
			protected.POST("/create/jbot/", ginJbot.CreateJbot(appContext.GetMainDBConnection()))
			protected.GET("/get/jbot/:id", ginJbot.GetJbotById(appContext.GetMainDBConnection()))
			protected.GET("/get/jbot/list", ginJbot.GetJbots(appContext.GetMainDBConnection()))
			protected.PUT("/update/jbot/:id", ginJbot.UpdateJbot(appContext.GetMainDBConnection()))
			protected.DELETE("/delete/jbot/:id", ginJbot.DeleteJbot(appContext.GetMainDBConnection()))
			protected.PUT("/update/password/jbot/:id", ginJbot.UpdateJbot(appContext.GetMainDBConnection()))

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
