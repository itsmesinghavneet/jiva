package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
)

func (s *Server) ListVolumes(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	apiContext.Write(&client.GenericCollection{
		Data: []interface{}{
			s.listVolumes(apiContext)[0],
		},
	})
	return nil
}

func (s *Server) GetVolume(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	v := s.getVolume(apiContext, id)
	if v == nil {
		rw.WriteHeader(http.StatusNotFound)
		return nil
	}

	apiContext.Write(v)
	return nil
}

func (s *Server) GetVolumeStats(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	stats := s.c.Stats()
	volumeStats := &VolumeStats{
		Resource:        client.Resource{Type: "stats"},
		RevisionCounter: stats.RevisionCounter,
		ReplicaCounter:  stats.ReplicaCounter,
		SCSIIOCount:     stats.SCSIIOCount,

		ReadIOPS:            strconv.FormatInt(stats.ReadIOPS, 10),
		TotalReadTime:       strconv.FormatInt(stats.TotalReadTime, 10),
		TotalReadBlockCount: strconv.FormatInt(stats.TotalReadBlockCount, 10),

		WriteIOPS:            strconv.FormatInt(stats.WriteIOPS, 10),
		TotalWriteTime:       strconv.FormatInt(stats.TotalWriteTime, 10),
		TotalWriteBlockCount: strconv.FormatInt(stats.TotalWriteBlockCount, 10),
	}
	apiContext.Write(volumeStats)
	return nil
}

func (s *Server) ShutdownVolume(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	v := s.getVolume(apiContext, id)
	if v == nil {
		rw.WriteHeader(http.StatusNotFound)
		return nil
	}

	if err := s.c.Shutdown(); err != nil {
		return err
	}

	return s.GetVolume(rw, req)
}

func (s *Server) RevertVolume(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	v := s.getVolume(apiContext, id)
	if v == nil {
		rw.WriteHeader(http.StatusNotFound)
		return nil
	}

	var input RevertInput
	if err := apiContext.Read(&input); err != nil {
		return err
	}

	if err := s.c.Revert(input.Name); err != nil {
		return err
	}

	return s.GetVolume(rw, req)
}

func (s *Server) ResizeVolume(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	v := s.getVolume(apiContext, id)
	if v == nil {
		rw.WriteHeader(http.StatusNotFound)
		return nil
	}

	var input ResizeInput
	if err := apiContext.Read(&input); err != nil {
		return err
	}

	if err := s.c.Resize(input.Name, input.Size); err != nil {
		return err
	}

	return s.GetVolume(rw, req)
}

func (s *Server) SnapshotVolume(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	v := s.getVolume(apiContext, id)
	if v == nil {
		rw.WriteHeader(http.StatusNotFound)
		return nil
	}

	var input SnapshotInput
	if err := apiContext.Read(&input); err != nil {
		return err
	}

	name, err := s.c.Snapshot(input.Name)
	if err != nil {
		return err
	}

	apiContext.Write(&SnapshotOutput{
		client.Resource{
			Id:   name,
			Type: "snapshotOutput",
		},
	})
	return nil
}

func (s *Server) StartVolume(rw http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)
	id := mux.Vars(req)["id"]

	v := s.getVolume(apiContext, id)
	if v == nil {
		rw.WriteHeader(http.StatusNotFound)
		return nil
	}

	var input StartInput
	if err := apiContext.Read(&input); err != nil {
		return err
	}

	if err := s.c.Start(input.Replicas...); err != nil {
		return err
	}

	return s.GetVolume(rw, req)
}

func (s *Server) listVolumes(context *api.ApiContext) []*Volume {
	return []*Volume{
		NewVolume(context, s.c.Name, len(s.c.ListReplicas())),
	}
}

func (s *Server) getVolume(context *api.ApiContext, id string) *Volume {
	for _, v := range s.listVolumes(context) {
		if v.Id == id {
			return v
		}
	}
	return nil
}