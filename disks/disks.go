package disks

import (
	"bufio"
	"os"
	"strings"
)

const (
	diskstatsPath = "/proc/diskstats"
	mountinfoPath = "/proc/self/mountinfo"
	blankStr      = ""
)

type Diskstats struct {
	Major  string
	Minor  string
	Device string
}

type Mountinfo struct {
	MountID     string
	ParentID    string
	MajMin      string
	Root        string
	Mountpoint  string
	FileSystem  string
	MountSource string
}

func mergeSpace(arr []byte) string {
	if len(arr) == 0 {
		return blankStr
	}

	newArr := make([]byte, 0)
	var preByte byte
	for _, b := range arr {
		if preByte == 32 && b == 32 {
			continue
		}
		newArr = append(newArr, b)
		preByte = b
	}

	return string(newArr)
}

func ReadDiskstats() ([]*Diskstats, error) {

	// read /proc/diskstats
	file, err := os.Open(diskstatsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	diskStatsSlice := make([]*Diskstats, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// read line
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		// merge extra spaces
		items := strings.Split(mergeSpace([]byte(line)), " ")

		if len(items) < 3 {
			continue
		}

		diskStats := &Diskstats{
			Major:  items[0],
			Minor:  items[1],
			Device: items[2],
		}
		diskStatsSlice = append(diskStatsSlice, diskStats)
	}

	return diskStatsSlice, nil
}

/*
The file contains lines of the form:

36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
(1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)

The numbers in parentheses are labels for the descriptions
below:

(1)  mount ID: a unique ID for the mount (may be reused after
	umount(2)).

(2)  parent ID: the ID of the parent mount (or of self for the
	root of this mount namespace's mount tree).

	If the parent mount point lies outside the process's root
	directory (see chroot(2)), the ID shown here won't have a
	corresponding record in mountinfo whose mount ID (field
	1) matches this parent mount ID (because mount points
	that lie outside the process's root directory are not
	shown in mountinfo).  As a special case of this point,
	the process's root mount point may have a parent mount
	(for the initramfs filesystem) that lies outside the
	process's root directory, and an entry for that mount
	point will not appear in mountinfo.

(3)  major:minor: the value of st_dev for files on this
	filesystem (see stat(2)).

(4)  root: the pathname of the directory in the filesystem
	which forms the root of this mount.

(5)  mount point: the pathname of the mount point relative to
	the process's root directory.

(6)  mount options: per-mount options.

(7)  optional fields: zero or more fields of the form
	"tag[:value]"; see below.

(8)  separator: the end of the optional fields is marked by a
	single hyphen.

(9)  filesystem type: the filesystem type in the form
	"type[.subtype]".

(10) mount source: filesystem-specific information or "none".

(11) super options: per-superblock options.
*/
func ReadMountInfo() ([]*Mountinfo, error) {
	file, err := os.Open(mountinfoPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	mountinfoSlice := make([]*Mountinfo, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// read line
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		// merge extra spaces
		items := strings.Split(mergeSpace([]byte(line)), " ")

		if len(items) < 11 {
			continue
		}

		mountinfo := &Mountinfo{
			MountID:     items[0],
			ParentID:    items[1],
			MajMin:      items[2],
			Root:        items[3],
			Mountpoint:  items[4],
			FileSystem:  items[8],
			MountSource: items[9],
		}
		mountinfoSlice = append(mountinfoSlice, mountinfo)
	}

	return mountinfoSlice, nil
}
