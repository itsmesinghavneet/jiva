package backupstore

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/yasker/backupstore/util"
)

type Volume struct {
	Name           string
	Driver         string
	Size           int64 `json:",string"`
	CreatedTime    string
	LastBackupName string
}

type Snapshot struct {
	Name        string
	CreatedTime string
}

type Backup struct {
	Name              string
	Driver            string
	VolumeName        string
	SnapshotName      string
	SnapshotCreatedAt string
	CreatedTime       string
	Size              int64 `json:",string"`

	Blocks     []BlockMapping `json:",omitempty"`
	SingleFile BackupFile     `json:",omitempty"`
}

var (
	backupstoreBase = "backupstore"
)

func SetBackupstoreBase(base string) {
	backupstoreBase = base
}

func GetBackupstoreBase() string {
	return backupstoreBase
}

func addVolume(volume *Volume, driver BackupStoreDriver) error {
	if volumeExists(volume.Name, driver) {
		return nil
	}

	if !util.ValidateName(volume.Name) {
		return fmt.Errorf("Invalid volume name %v", volume.Name)
	}

	if err := saveVolume(volume, driver); err != nil {
		log.Error("Fail add volume ", volume.Name)
		return err
	}
	log.Debug("Added backupstore volume ", volume.Name)

	return nil
}

func removeVolume(volumeName string, driver BackupStoreDriver) error {
	if !util.ValidateName(volumeName) {
		return fmt.Errorf("Invalid volume name %v", volumeName)
	}

	if !volumeExists(volumeName, driver) {
		return fmt.Errorf("Volume %v doesn't exist in backupstore", volumeName)
	}

	volumeDir := getVolumePath(volumeName)
	if err := driver.Remove(volumeDir); err != nil {
		return err
	}
	log.Debug("Removed volume directory in backupstore: ", volumeDir)
	log.Debug("Removed backupstore volume ", volumeName)

	return nil
}

func encodeBackupURL(backupName, volumeName, destURL string) string {
	v := url.Values{}
	v.Add("volume", volumeName)
	v.Add("backup", backupName)
	return destURL + "?" + v.Encode()
}

func decodeBackupURL(backupURL string) (string, string, error) {
	u, err := url.Parse(backupURL)
	if err != nil {
		return "", "", err
	}
	v := u.Query()
	volumeName := v.Get("volume")
	backupName := v.Get("backup")
	if !util.ValidateName(volumeName) || !util.ValidateName(backupName) {
		return "", "", fmt.Errorf("Invalid name parsed, got %v and %v", backupName, volumeName)
	}
	return backupName, volumeName, nil
}

func addListVolume(resp map[string]map[string]string, volumeName string, driver BackupStoreDriver, storageDriverName string) error {
	if volumeName == "" {
		return fmt.Errorf("Invalid empty volume Name")
	}

	if !util.ValidateName(volumeName) {
		return fmt.Errorf("Invalid volume name %v", volumeName)
	}

	backupNames, err := getBackupNamesForVolume(volumeName, driver)
	if err != nil {
		return err
	}

	volume, err := loadVolume(volumeName, driver)
	if err != nil {
		return err
	}
	//Skip any volumes not owned by specified storage driver
	if volume.Driver != storageDriverName {
		return nil
	}

	for _, backupName := range backupNames {
		backup, err := loadBackup(backupName, volumeName, driver)
		if err != nil {
			return err
		}
		r := fillBackupInfo(backup, volume, driver.GetURL())
		resp[r["BackupURL"]] = r
	}
	return nil
}

func List(volumeName, destURL, storageDriverName string) (map[string]map[string]string, error) {
	driver, err := GetBackupStoreDriver(destURL)
	if err != nil {
		return nil, err
	}
	resp := make(map[string]map[string]string)
	if volumeName != "" {
		if err = addListVolume(resp, volumeName, driver, storageDriverName); err != nil {
			return nil, err
		}
	} else {
		volumeNames, err := getVolumeNames(driver)
		if err != nil {
			return nil, err
		}
		for _, volumeName := range volumeNames {
			if err := addListVolume(resp, volumeName, driver, storageDriverName); err != nil {
				return nil, err
			}
		}
	}
	return resp, nil
}

func fillBackupInfo(backup *Backup, volume *Volume, destURL string) map[string]string {
	return map[string]string{
		"BackupName":        backup.Name,
		"BackupURL":         encodeBackupURL(backup.Name, backup.VolumeName, destURL),
		"DriverName":        volume.Driver,
		"VolumeName":        backup.VolumeName,
		"VolumeSize":        strconv.FormatInt(volume.Size, 10),
		"VolumeCreatedAt":   volume.CreatedTime,
		"SnapshotName":      backup.SnapshotName,
		"SnapshotCreatedAt": backup.SnapshotCreatedAt,
		"CreatedTime":       backup.CreatedTime,
		"Size":              strconv.FormatInt(backup.Size, 10),
	}
}

func GetBackupInfo(backupURL string) (map[string]string, error) {
	driver, err := GetBackupStoreDriver(backupURL)
	if err != nil {
		return nil, err
	}
	backupName, volumeName, err := decodeBackupURL(backupURL)
	if err != nil {
		return nil, err
	}

	volume, err := loadVolume(volumeName, driver)
	if err != nil {
		return nil, err
	}

	backup, err := loadBackup(backupName, volumeName, driver)
	if err != nil {
		return nil, err
	}
	return fillBackupInfo(backup, volume, driver.GetURL()), nil
}

func LoadVolume(backupURL string) (*Volume, error) {
	_, volumeName, err := decodeBackupURL(backupURL)
	if err != nil {
		return nil, err
	}
	driver, err := GetBackupStoreDriver(backupURL)
	if err != nil {
		return nil, err
	}
	return loadVolume(volumeName, driver)
}
