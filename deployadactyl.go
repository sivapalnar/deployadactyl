package deployadactyl

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/geterrors"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	basicAuthHeaderNotFound = "basic auth header not found"
	invalidPostRequest      = "invalid POST request"
	cannotDeployApplication = "cannot deploy application"
)

type Deployadactyl struct {
	Deployer     I.Deployer
	Log          *logging.Logger
	Config       config.Config
	EventManager I.EventManager
	Randomizer   I.Randomizer
}

func (d *Deployadactyl) Deploy(c *gin.Context) {
	d.Log.Debug("Request originated from: %+v", c.Request.RemoteAddr)

	var (
		environment            = c.Param("environment")
		authenticationRequired = d.Config.Environments[environment].Authenticate
		buffer                 = &bytes.Buffer{}
	)

	username, password, ok := c.Request.BasicAuth()

	if !ok {
		if authenticationRequired {
			err := errors.New(basicAuthHeaderNotFound)
			logError(d.Log, err)
			c.Writer.WriteHeader(401)
			c.Writer.WriteString(err.Error())
			return
		}
		username = d.Config.Username
		password = d.Config.Password
	}

	deploymentInfo, err := getDeploymentInfo(c.Request.Body)
	deploymentInfo.Username = username
	deploymentInfo.Password = password
	deploymentInfo.Environment = environment
	deploymentInfo.Org = c.Param("org")
	deploymentInfo.Space = c.Param("space")
	deploymentInfo.AppName = c.Param("appName")
	deploymentInfo.UUID = d.Randomizer.StringRunes(128)

	d.Log.Debug("Deployment properties:\n\tartifact url: %+v", deploymentInfo.ArtifactURL)
	if err != nil {
		logError(d.Log, errors.WrapPrefix(err, invalidPostRequest, 0))
		c.Writer.WriteHeader(500)
		c.Writer.WriteString(err.Error())
		return
	}

	deployEventData := S.DeployEventData{
		Writer:         buffer,
		DeploymentInfo: &deploymentInfo,
		RequestBody:    c.Request.Body,
	}

	m, err := base64.StdEncoding.DecodeString(deploymentInfo.Manifest)
	if err != nil {
		logError(d.Log, errors.WrapPrefix(err, invalidPostRequest, 0))
		c.Writer.WriteHeader(500)
		c.Writer.WriteString(err.Error())
		return
	}

	deploymentInfo.Manifest = string(m)

	deployStartEvent := S.Event{
		Type: "deploy.start",
		Data: deployEventData,
	}

	err = d.EventManager.Emit(deployStartEvent)
	if err != nil {
		fmt.Fprint(buffer, err)
		return
	}

	err = d.Deployer.Deploy(deploymentInfo, buffer)
	if err != nil {
		logError(d.Log, errors.WrapPrefix(err, cannotDeployApplication, 0))
		c.Writer.WriteHeader(500)
		c.Error(err)
	}

	io.Copy(c.Writer, buffer)
}

func getDeploymentInfo(reader io.Reader) (S.DeploymentInfo, error) {
	deploymentInfo := S.DeploymentInfo{}
	err := json.NewDecoder(reader).Decode(&deploymentInfo)
	if err != nil {
		return deploymentInfo, err
	}

	getter := geterrors.WrapFunc(func(key string) string {
		if key == "artifact_url" {
			return deploymentInfo.ArtifactURL
		}
		return ""
	})
	getter.Get("artifact_url")

	err = getter.Err("The following properties are missing")
	if err != nil {
		return S.DeploymentInfo{}, err
	}
	return deploymentInfo, nil
}

func logError(log *logging.Logger, err error) {
	if wrappedErr, ok := err.(*errors.Error); ok {
		log.Error(wrappedErr.ErrorStack())
		return
	}
	log.Error(err.Error())
}