package app

import (
	appContext "j_ai_trade/components/app_context"
	"j_ai_trade/middlewares"
	ginAuth "j_ai_trade/modules/auth/transport/gin"
	ginOrder "j_ai_trade/modules/order/transport/gin"
	ginUser "j_ai_trade/modules/user/transport/gin"
	"j_ai_trade/telegram"
	ginTelegram "j_ai_trade/telegram/transport/gin"

	"github.com/gin-contrib/cors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitializeApp(appContext appContext.AppContext) {
	router := appContext.GetGinApp()

	// Swagger
	swaggerGroup := router.Group("/docs")
	swaggerGroup.Use(BasicAuthMiddleware())
	swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Global middleware
	router.Use(middlewares.PanicRecoveryMiddleware())
	router.Use(middlewares.RequestIDMiddleware())
	router.Use(middlewares.LoggerMiddleware())

	v1 := router.Group("/api/v1")

	// ===== Public auth routes (with dedicated rate limit) =====
	{
		auth := v1.Group("/auth")
		auth.Use(middlewares.AuthRateLimitMiddleware())
		{
			auth.POST("/register", ginAuth.Register(appContext.GetMainDBConnection()))
			auth.POST("/send-email-registration-code", ginAuth.SendEmailRegistrationCode(appContext.GetMainDBConnection()))
			auth.POST("/verify-email-registration-code", ginAuth.EmailRegistrationCodeVerification(appContext.GetMainDBConnection()))

			auth.POST("/login", ginAuth.Login(appContext.GetMainDBConnection()))

			auth.POST("/send-forgot-password-code", ginAuth.SendForgotPasswordCode(appContext.GetMainDBConnection()))
			auth.POST("/verify-reset-password-code", ginAuth.VerifyResetPasswordCode(appContext.GetMainDBConnection()))
			auth.POST("/reset-password", ginAuth.ResetPassword(appContext.GetMainDBConnection()))

			auth.POST("/refresh-token", ginAuth.RefreshToken())
		}
	}

	// ===== Telegram =====
	{
		telegramService := telegram.NewTelegramService()
		v1.POST("/telegram/send", ginTelegram.SendTelegramMessage(telegramService))
	}

	// ===== Protected routes =====
	protected := v1.Group("")
	protected.Use(middlewares.AuthMiddleware())

	// User
	{
		protected.POST("/user/create", ginUser.CreateUser(appContext.GetMainDBConnection()))
		protected.GET("/user/get/:id", ginUser.GetUserById(appContext.GetMainDBConnection()))
		protected.GET("/user/list", ginUser.GetUsers(appContext.GetMainDBConnection()))
		protected.PUT("/user/update/:id", ginUser.UpdateUser(appContext.GetMainDBConnection()))
		protected.DELETE("/user/delete/:id", ginUser.DeleteUser(appContext.GetMainDBConnection()))
		protected.PUT("/user/password/update/:id", ginUser.UpdateUserPassword(appContext.GetMainDBConnection()))
	}

	// Order (reserved for storing ensemble signals + backtest logs)
	{
		protected.POST("/order/create", ginOrder.CreateOrder(appContext.GetMainDBConnection()))
		protected.GET("/order/get/:id", ginOrder.GetOrderById(appContext.GetMainDBConnection()))
		protected.GET("/order/list", ginOrder.GetOrders(appContext.GetMainDBConnection()))
		protected.PUT("/order/update/:id", ginOrder.UpdateOrder(appContext.GetMainDBConnection()))
		protected.DELETE("/order/delete/:id", ginOrder.DeleteOrder(appContext.GetMainDBConnection()))
	}
}
