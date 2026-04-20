package app

import (
	appContext "j_ai_trade/components/app_context"
	"j_ai_trade/middlewares"
	ginAiExpert "j_ai_trade/modules/ai_expert/transport/gin"
	ginApiKey "j_ai_trade/modules/api_key/transport/gin"
	ginAuth "j_ai_trade/modules/auth/transport/gin"
	ginFutures "j_ai_trade/modules/futures"
	ginJbot "j_ai_trade/modules/j_bot/transport/gin"
	ginOrder "j_ai_trade/modules/order/transport/gin"
	ginOtp "j_ai_trade/modules/otp/transport/gin"
	ginPermission "j_ai_trade/modules/permission/transport/gin"
	ginSignal "j_ai_trade/modules/signal/transport/gin"
	ginSubscription "j_ai_trade/modules/subscription/transport/gin"
	ginUser "j_ai_trade/modules/user/transport/gin"
	"j_ai_trade/telegram"
	ginTelegram "j_ai_trade/telegram/transport/gin"

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
			//Register
			auth.POST("/register", ginAuth.Register(appContext.GetMainDBConnection()))
			auth.POST("/send-email-registration-code", ginAuth.SendEmailRegistrationCode(appContext.GetMainDBConnection()))
			auth.POST("/verify-email-registration-code", ginAuth.EmailRegistrationCodeVerification(appContext.GetMainDBConnection()))

			//Login
			auth.POST("/login", ginAuth.Login(appContext.GetMainDBConnection()))

			// Forgot Password
			auth.POST("/send-forgot-password-code", ginAuth.SendForgotPasswordCode(appContext.GetMainDBConnection()))
			auth.POST("/verify-reset-password-code", ginAuth.VerifyResetPasswordCode(appContext.GetMainDBConnection()))
			auth.POST("/reset-password", ginAuth.ResetPassword(appContext.GetMainDBConnection()))

			// Refresh Token (no authentication required)
			auth.POST("/refresh-token", ginAuth.RefreshToken())

		}
	}

	{
		tool := v1.Group("/tool")
		{
			tool.POST("/futures/leverage", ginFutures.CalculateLeverageAPI())
		}
	}

	// Telegram API
	{
		telegramService := telegram.NewTelegramService()
		{
			v1.Group("").POST("/telegram/send", ginTelegram.SendTelegramMessage(telegramService))
		}
	}

	//API configs
	protected := v1.Group("")
	protected.Use(middlewares.AuthMiddleware())

	//Protected Auth
	// User API
	{
		{
			protected.POST("/user/create", ginUser.CreateUser(appContext.GetMainDBConnection()))
			protected.GET("/user/get/:id", ginUser.GetUserById(appContext.GetMainDBConnection()))
			protected.GET("/user/list", ginUser.GetUsers(appContext.GetMainDBConnection()))
			protected.PUT("/user/update/:id", ginUser.UpdateUser(appContext.GetMainDBConnection()))
			protected.DELETE("/user/delete/:id", ginUser.DeleteUser(appContext.GetMainDBConnection()))
			protected.PUT("/user/password/update/:id", ginUser.UpdateUserPassword(appContext.GetMainDBConnection()))
		}
	}

	// Subscription API
	{
		{
			protected.POST("/subscription/create", ginSubscription.CreateSubscription(appContext.GetMainDBConnection()))
			protected.GET("/subscription/get/:id", ginSubscription.GetSubscriptionById(appContext.GetMainDBConnection()))
			protected.GET("/subscription/list", ginSubscription.GetSubscriptions(appContext.GetMainDBConnection()))
			protected.PUT("/subscription/update/:id", ginSubscription.UpdateSubscription(appContext.GetMainDBConnection()))
			protected.DELETE("/subscription/delete/:id", ginSubscription.DeleteSubscription(appContext.GetMainDBConnection()))
		}
	}

	// ApiKey API
	{
		{
			protected.POST("/api-key/create", ginApiKey.CreateApiKey(appContext.GetMainDBConnection()))
			protected.GET("/api-key/get/:id", ginApiKey.GetApiKeyById(appContext.GetMainDBConnection()))
			protected.GET("/api-key/list", ginApiKey.GetApiKeys(appContext.GetMainDBConnection()))
			protected.PUT("/api-key/update/:id", ginApiKey.UpdateApiKey(appContext.GetMainDBConnection()))
			protected.DELETE("/api-key/delete/:id", ginApiKey.DeleteApiKey(appContext.GetMainDBConnection()))
		}
	}

	// Permission API
	{
		{
			protected.POST("/permission/create", ginPermission.CreatePermission(appContext.GetMainDBConnection()))
			protected.GET("/permission/get/:id", ginPermission.GetPermissionById(appContext.GetMainDBConnection()))
			protected.GET("/permission/list", ginPermission.GetPermissions(appContext.GetMainDBConnection()))
			protected.PUT("/permission/update/:id", ginPermission.UpdatePermission(appContext.GetMainDBConnection()))
			protected.DELETE("/permission/delete/:id", ginPermission.DeletePermission(appContext.GetMainDBConnection()))
		}
	}

	// AiExpert API
	{
		{
			protected.POST("/ai-expert/create", ginAiExpert.CreateAiExpert(appContext.GetMainDBConnection()))
			protected.GET("/ai-expert/get/:id", ginAiExpert.GetAiExpertById(appContext.GetMainDBConnection()))
			protected.GET("/ai-expert/list", ginAiExpert.GetAiExperts(appContext.GetMainDBConnection()))
			protected.PUT("/ai-expert/update/:id", ginAiExpert.UpdateAiExpert(appContext.GetMainDBConnection()))
			protected.DELETE("/ai-expert/delete/:id", ginAiExpert.DeleteAiExpert(appContext.GetMainDBConnection()))
		}
	}

	// Order API
	{
		{
			protected.POST("/order/create", ginOrder.CreateOrder(appContext.GetMainDBConnection()))
			protected.GET("/order/get/:id", ginOrder.GetOrderById(appContext.GetMainDBConnection()))
			protected.GET("/order/list", ginOrder.GetOrders(appContext.GetMainDBConnection()))
			protected.PUT("/order/update/:id", ginOrder.UpdateOrder(appContext.GetMainDBConnection()))
			protected.DELETE("/order/delete/:id", ginOrder.DeleteOrder(appContext.GetMainDBConnection()))
		}
	}

	// Signal API
	{
		{
			protected.POST("/signal/create", ginSignal.CreateSignal(appContext.GetMainDBConnection()))
			protected.GET("/signal/get/:id", ginSignal.GetSignalById(appContext.GetMainDBConnection()))
			protected.GET("/signal/list", ginSignal.GetSignals(appContext.GetMainDBConnection()))
			protected.PUT("/signal/update/:id", ginSignal.UpdateSignal(appContext.GetMainDBConnection()))
			protected.DELETE("/signal/delete/:id", ginSignal.DeleteSignal(appContext.GetMainDBConnection()))
		}
	}

	// Otp API
	{
		{
			protected.POST("/otp/create", ginOtp.CreateOtp(appContext.GetMainDBConnection()))
			protected.GET("/otp/get/:id", ginOtp.GetOtpById(appContext.GetMainDBConnection()))
			protected.GET("/otp/list", ginOtp.GetOtps(appContext.GetMainDBConnection()))
			protected.PUT("/otp/update/:id", ginOtp.UpdateOtp(appContext.GetMainDBConnection()))
			protected.DELETE("/otp/delete/:id", ginOtp.DeleteOtp(appContext.GetMainDBConnection()))
		}
	}

	// JBot API
	{
		{
			protected.POST("/jbot/create", ginJbot.CreateJbot(appContext.GetMainDBConnection()))
			protected.GET("/jbot/get/:id", ginJbot.GetJbotById(appContext.GetMainDBConnection()))
			protected.GET("/jbot/list", ginJbot.GetJbots(appContext.GetMainDBConnection()))
			protected.PUT("/jbot/update/:id", ginJbot.UpdateJbot(appContext.GetMainDBConnection()))
			protected.DELETE("/jbot/delete/:id", ginJbot.DeleteJbot(appContext.GetMainDBConnection()))
			protected.PUT("/jbot/password/update/:id", ginJbot.UpdateJbot(appContext.GetMainDBConnection()))
		}
	}

}
