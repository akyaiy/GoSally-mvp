package run_manager

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
)

func File(index string) RunFileManagerContract {
	value, ok := indexedPaths[index]
	if !ok {
		err := indexPaths()
		if err != nil {
			return &RunFileManager{
				err: err,
			}
		}
		value, ok = indexedPaths[index]
		if !ok {
			return &RunFileManager{
				err: fmt.Errorf("cannot detect file under index %s", index),
			}
		}
	}
	return &RunFileManager{
		indexedPath: value,
	}
}

func (r *RunFileManager) Open() (*os.File, error) {
	if r.err != nil {
		return nil, r.err
	}
	file, err := os.OpenFile(r.indexedPath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	r.file = file
	return file, nil
}

func (r *RunFileManager) Close() error {
	return r.file.Close()
}

func (r *RunFileManager) Watch(parentCtx context.Context, callback func()) (context.CancelFunc, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.file == nil {
		return nil, fmt.Errorf("file is not opened")
	}

	info, err := r.file.Stat()
	if err != nil {
		return nil, err
	}
	origStat := info.Sys().(*syscall.Stat_t)
	origIno := origStat.Ino
	origModTime := info.ModTime()

	ctx, cancel := context.WithCancel(parentCtx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				newInfo, err := os.Stat(r.indexedPath)
				if err != nil {
					if os.IsNotExist(err) {
						callback()
						return
					}
				} else {
					newStat := newInfo.Sys().(*syscall.Stat_t)
					if newStat.Ino != origIno {
						callback()
						return
					}
					if !newInfo.ModTime().Equal(origModTime) {
						callback()
						return
					}
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return cancel, nil
}
