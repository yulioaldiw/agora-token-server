package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/AgoraIO-Community/go-tokenbuilder/rtctokenbuilder"
	"github.com/AgoraIO-Community/go-tokenbuilder/rtmtokenbuilder"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var appID, appCertificate string

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	appIDEnv, appIDExists := os.LookupEnv("APP_ID")
	appCertEnv, appCertExists := os.LookupEnv("APP_CERTIFICATE")

	if !appIDExists || !appCertExists {
		log.Fatal("FATAL ERROR: ENV not properly configured, check APP_ID and APP_CERTIFICATE")
	} else {
		appID = appIDEnv
		appCertificate = appCertEnv
	}

	api := gin.Default()

	api.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	api.GET("rtc/:channelName/:role/:tokenType/:uid/", getRTCToken)
	api.GET("rtm/:uid/", getRTMToken)
	api.GET("rte/:channelName/:role/:tokenType/:uid/", getBothTokens)

	api.Run(":8080")
}

func getRTCToken(c *gin.Context) {
	// get param values
	channelName, tokenType, uidString, role, expireTimestamp, err := parseRTCParams(c)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": "Error generating RTC token: " + err.Error(),
		})

		return
	}

	// generate the token
	rtcToken, tokenErr := generateRTCToken(channelName, uidString, tokenType, role, expireTimestamp)

	// return the token in JSON response
	if tokenErr != nil {
		log.Println(tokenErr)
		c.Error(err)
		c.AbortWithStatusJSON(400, gin.H{
			"status": 400,
			"error":  "Error generating RTC token: " + tokenErr.Error(),
		})
	} else {
		c.JSON(200, gin.H{
			"rtcToken": rtcToken,
		})
	}
}

func getRTMToken(c *gin.Context) {
	// get param values
	uidString, expireTimestamp, err := parseRTMParams(c)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": "Error generating RTM token",
		})

		return
	}

	//build RTM token
	rtmToken, tokenErr := rtmtokenbuilder.BuildToken(appID, appCertificate, uidString, rtmtokenbuilder.RoleRtmUser, expireTimestamp)

	// return RTM token
	if tokenErr != nil {
		log.Println(err)
		c.Error(err)
		c.AbortWithStatusJSON(400, gin.H{
			"status": 400,
			"error":  "Error generating RTM token: " + tokenErr.Error(),
		})
	} else {
		c.JSON(200, gin.H{
			"rtmToken": rtmToken,
		})
	}
}

func getBothTokens(c *gin.Context) {
	// get the params
	channelName, tokenType, uidString, role, expireTimestamp, rtcParamErr := parseRTCParams(c)
	if rtcParamErr != nil {
		c.Error(rtcParamErr)
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": "Error generating tokens: " + rtcParamErr.Error(),
		})
	}

	// generate rtc token
	rtcToken, rtcTokenErr := generateRTCToken(channelName, uidString, tokenType, role, expireTimestamp)

	// generate rtm token
	rtmToken, rtmTokenErr := rtmtokenbuilder.BuildToken(appID, appCertificate, uidString, rtmtokenbuilder.RoleRtmUser, expireTimestamp)

	// return both tokens
	if rtcTokenErr != nil {
		c.Error(rtcTokenErr)
		errMsg := "Error generating RTC token: " + rtcTokenErr.Error()
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": errMsg,
		})
	} else if rtmTokenErr != nil {
		c.Error(rtmTokenErr)
		errMsg := "Error generating RTM token: " + rtmTokenErr.Error()
		c.AbortWithStatusJSON(400, gin.H{
			"status":  400,
			"message": errMsg,
		})
	} else {
		c.JSON(200, gin.H{
			"rtcToken": rtcToken,
			"rtmToken": rtmToken,
		})
	}
}

func parseRTCParams(c *gin.Context) (channelName, tokenType, uidString string, role rtctokenbuilder.Role, expireTimestamp uint32, err error) {
	// get param values
	channelName = c.Param("channelName")
	roleString := c.Param("role")
	tokenType = c.Param("tokenType")
	uidString = c.Param("uid")
	expireTime := c.DefaultQuery("expiry", "3600")

	if roleString == "publisher" {
		role = rtctokenbuilder.RolePublisher
	} else {
		role = rtctokenbuilder.RoleSubscriber
	}

	expireTime64, parseErr := strconv.ParseUint(expireTime, 10, 64)
	if parseErr != nil {
		err = fmt.Errorf("failed to parse expireTime: %s, causing error: %s", expireTime, parseErr)
	}

	expireTimeSeconds := uint32(expireTime64)
	currentTimestamp := uint32(time.Now().UTC().Unix())
	expireTimestamp = currentTimestamp + expireTimeSeconds

	// return
	return channelName, tokenType, uidString, role, expireTimestamp, err
}

func parseRTMParams(c *gin.Context) (uidString string, expireTimestamp uint32, err error) {
	// get param values
	uidString = c.Param("uid")
	expireTime := c.DefaultQuery("expiry", "3600")

	expireTime64, parseErr := strconv.ParseUint(expireTime, 10, 64)
	if parseErr != nil {
		err = fmt.Errorf("failed to parse expireTime: %s, causing error: %s", expireTime, parseErr)
	}

	expireTimeSeconds := uint32(expireTime64)
	currentTimestamp := uint32(time.Now().UTC().Unix())
	expireTimestamp = currentTimestamp + expireTimeSeconds

	return uidString, expireTimestamp, err
}

func generateRTCToken(channelName, uidString, tokenType string, role rtctokenbuilder.Role, expireTimestamp uint32) (rtcToken string, err error) {
	// check token type
	if tokenType == "userAccount" {
		rtcToken, err = rtctokenbuilder.BuildTokenWithUserAccount(appID, appCertificate, channelName, uidString, role, expireTimestamp)

		return rtcToken, err
	} else if tokenType == "uid" {
		uid64, parseErr := strconv.ParseUint(uidString, 10, 64)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse uidString: %s, to uint causing error: %s", uidString, parseErr)

			return "", err
		}

		uid := uint32(uid64)
		rtcToken, err = rtctokenbuilder.BuildTokenWithUID(appID, appCertificate, channelName, uid, role, expireTimestamp)

		return rtcToken, err
	} else {
		err = fmt.Errorf("failed to generate RTC token for unknown tokenType: %s", tokenType)
		log.Println(err)

		return "", err
	}
}
