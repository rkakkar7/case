package microservice

import (
	"fmt"
	"math/rand"

	"case/common"
	"case/common/constants"
	"case/common/types"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/syncmap"
)

type Controller struct {
	services    syncmap.Map // unique service, e.g. key - localhost:9081, value - service
	serviceList syncmap.Map // list of same services, e.g. key - userserver, value - []services
}

func NewController() *Controller {
	con := &Controller{}
	return con
}

// Add a service to controller
func (con *Controller) AddService(serviceDef types.ServiceDefinition, routerID, callerService string) bool {
	log.Infof("AddService: %s %s_%d", serviceDef.Service, serviceDef.Host, serviceDef.Port)
	serviceID := fmt.Sprintf("%s_%d", serviceDef.Host, serviceDef.Port)
	srv := NewService(serviceDef.Service, callerService, serviceDef.Host, serviceDef.Port, routerID)
	srv.Start()
	con.services.Store(serviceID, srv)
	log.Infof("service.Store: %s", serviceID)
	serviceList, ok := con.serviceList.Load(srv.Name)
	if !ok {
		serviceList = make([]*service, 0)
	}
	serviceList = append(serviceList.([]*service), srv)
	con.serviceList.Store(serviceDef.Service, serviceList)
	log.Infof("serviceList.Store: %+v", serviceDef.Service)
	return true
}

// Get a service corresponding to user and service name
func (con *Controller) GetService(serviceName, userID string) *service {
	log.Infof("GetService: %s", serviceName)
	srvcList, ok := con.serviceList.Load(serviceName)
	if !ok {
		log.Errorf("get-service: no service of type %s", serviceName)
		return nil
	}

	srvcListSlice := srvcList.([]*service)
	numService := len(srvcListSlice)
	if numService > 0 {
		if numService == 1 {
			return srvcListSlice[0]
		}

		if userID != "" {
			id := con.getHash(userID) % uint64(len(srvcListSlice))
			return srvcListSlice[id]
		}

		return srvcListSlice[0]
	}

	log.Errorf("get-service: all service of type %s removed", serviceName)
	return nil
}

// Get a random service for a list of same services
func (con *Controller) GetRandomService(serviceName string) *service {
	log.Infof("GetRandomService: %s", serviceName)
	srvcList, ok := con.services.Load(serviceName)
	if !ok {
		log.Errorf("get-service: no service of type %s", serviceName)
		return nil
	}

	srvcListSlice := srvcList.([]*service)
	numService := len(srvcListSlice)
	if numService > 0 {
		if numService == 1 {
			return srvcListSlice[0]
		}
		return srvcListSlice[rand.Intn(len(srvcListSlice))]
	}

	log.Errorf("GetRandomService: all service of type %s removed", serviceName)
	return nil
}

func (con *Controller) GetServiceFromID(serviceID string) *service {
	val, ok := con.services.Load(serviceID)
	if ok {
		srvc := val.(*service)
		return srvc
	}
	return nil
}

func (con *Controller) getHash(key string) uint64 {
	return common.FNV64([]byte(key))
}

func (con *Controller) SendMessage(msg types.Message) (chan types.Message, error) {
	switch msg.RouterHeader {
	case constants.RouterHeader.User:
		srvc := con.GetService(constants.UserService, msg.UserID)
		if srvc == nil || srvc.Send == nil {
			return nil, fmt.Errorf("no userservice present")
		}
		key := uuid.NewV4().String()
		ch := make(chan types.Message)
		srvc.chanMap.Store(key, ch)
		msg.ChannelKey = key
		srvc.Send(msg)
		log.Infof("SendMessage user %v", srvc.Name)
		return ch, nil
	}
	return nil, fmt.Errorf("wrong request")
}

func (con *Controller) RemoveAllService() {
	con.serviceList.Range(func(key, value interface{}) bool {
		srvcListSlice := value.([]*service)
		for _, srvc := range srvcListSlice {
			srvc.Stop()
		}
		return true
	})
}

func (con *Controller) RemoveService(serviceID, serviceName string) {
	con.services.Delete(serviceID)

	serviceList, ok := con.serviceList.Load(serviceName)
	if !ok {
		return
	}

	latestServiceList := make([]*service, 0)
	serviceListSlice := serviceList.([]*service)
	for _, srv := range serviceListSlice {
		if srv.ID != serviceID {
			latestServiceList = append(latestServiceList, srv)
		}
	}
	con.serviceList.Store(serviceName, latestServiceList)
}
