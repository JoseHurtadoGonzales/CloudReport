package api

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"

	"github.com/cloudreport/api/internal/auth"
	"github.com/cloudreport/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// AssetUploadHandler accepts a multipart/form-data POST with one or more files
// in the `files` field (or a single `file`), stores them in SeaweedFS, and
// creates an asset row.
//
// Optional form fields:
//
//	name      asset display name (defaults to file basename)
//	folder    folder shortid to place the asset in
//	mimeType  override the detected MIME type
func AssetUploadHandler(d *Deps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		files := form.File["files"]
		if len(files) == 0 {
			if single, err := c.FormFile("file"); err == nil {
				files = []*multipart.FileHeader{single}
			}
		}
		if len(files) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "no files provided (field name: 'files' or 'file')"})
		}

		u := auth.UserFrom(c)
		spec := store.EntitySpecs["assets"]
		var results []fiber.Map

		for _, fh := range files {
			fp, err := fh.Open()
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			data, err := io.ReadAll(fp)
			fp.Close()
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}

			mime := fh.Header.Get("Content-Type")
			if explicit := firstValue(form.Value["mimeType"]); explicit != "" {
				mime = explicit
			}
			if mime == "" {
				mime = http.DetectContentType(data)
			}

			name := firstValue(form.Value["name"])
			if name == "" {
				name = fh.Filename
			}

			key, err := d.Blob.Put(c.UserContext(), "assets", "", data, mime)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("upload: %s", err)})
			}

			payload := map[string]any{
				"name":       name,
				"blob_key":   key,
				"mime_type":  mime,
				"size_bytes": len(data),
			}
			if folderShortid := firstValue(form.Value["folder"]); folderShortid != "" {
				if f, err := d.DB.GetGenericByShortid(c.UserContext(), store.EntitySpecs["folders"], folderShortid); err == nil {
					payload["folder_id"] = f["id"]
				}
			}
			if u != nil {
				payload["owner_id"] = u.ID
			}
			row, err := d.DB.InsertGeneric(c.UserContext(), spec, payload)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			results = append(results, fiber.Map{
				"asset":    row,
				"filename": path.Base(name),
				"size":     len(data),
				"mimeType": mime,
				"blobKey":  key,
			})
		}

		return c.Status(201).JSON(fiber.Map{"uploaded": results})
	}
}

func firstValue(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
