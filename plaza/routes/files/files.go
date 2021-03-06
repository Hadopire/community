/*
 * Nanocloud Community, a comprehensive platform to turn any application
 * into a cloud solution.
 *
 * Copyright (C) 2016 Nanocloud Software
 *
 * This file is part of Nanocloud community.
 *
 * Nanocloud community is free software; you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * Nanocloud community is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package files

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
)

type hash map[string]interface{}

type file_t struct {
	Id         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

func Post(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query()["username"][0]
	filename := r.URL.Query()["filename"][0]

	dst, err := os.Create(fmt.Sprintf("/home/%s/%s", username, filename))
	if err != nil {
		log.Error(err)
		http.Error(w, "Unable to create destination file", http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	_, err = io.Copy(dst, r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, "Unable to write destination file", http.StatusInternalServerError)
		return
	}

}

func Get(c *echo.Context) error {
	filepath := c.Query("path")
	showHidden := c.Query("show_hidden") == "true"
	create := c.Query("create") == "true"

	if len(filepath) < 1 {
		return c.JSON(
			http.StatusBadRequest,
			hash{
				"error": "Path not specified",
			},
		)
	}

	s, err := os.Stat(filepath)
	if err != nil {
		log.Error(err.(*os.PathError).Err.Error())
		m := err.(*os.PathError).Err.Error()
		if m == "no such file or directory" || m == "The system cannot find the file specified." {
			if create {
				err := os.MkdirAll(filepath, 0777)
				if err != nil {
					return err
				}
				s, err = os.Stat(filepath)
				if err != nil {
					return err
				}
			} else {
				return c.JSON(
					http.StatusNotFound,
					hash{
						"error": "no such file or directory",
					},
				)
			}
		} else {
			return err
		}
	}

	if s.Mode().IsDir() {
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer f.Close()

		files, err := f.Readdir(-1)
		if err != nil {
			return err
		}

		rt := make([]file_t, 0)

		for _, file := range files {
			name := file.Name()
			if !showHidden && isFileHidden(file) {
				continue
			}

			fullpath := path.Join(filepath, name)
			id, err := loadFileId(fullpath)
			if err != nil {
				log.Errorf("Cannot retrieve file id for file: %s: %s", fullpath, err.Error())
				continue
			}

			f := file_t{
				Id:   id,
				Type: "file",
			}

			attr := make(map[string]interface{}, 0)
			f.Attributes = attr

			attr["mod_time"] = file.ModTime().Unix()
			attr["size"] = file.Size()
			attr["name"] = name

			if file.IsDir() {
				attr["type"] = "directory"
			} else {
				attr["type"] = "regular file"
			}
			rt = append(rt, f)
		}
		/*
		 * The Content-Length is not set is the buffer length is more than 2048
		 */
		b, err := json.Marshal(hash{
			"data": rt,
		})
		if err != nil {
			return err
		}

		r := c.Response()
		r.Header().Set("Content-Length", strconv.Itoa(len(b)))
		r.Header().Set("Content-Type", "application/json; charset=utf-8")
		r.Write(b)
		return nil
	}

	return c.File(
		filepath,
		s.Name(),
		true,
	)
}
