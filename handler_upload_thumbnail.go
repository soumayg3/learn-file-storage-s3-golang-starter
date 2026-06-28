package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxUpload = 10 << 20

	if err := r.ParseMultipartForm(maxUpload); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	fCt := header.Header.Get("Content-Type")
	if mt, _, err := mime.ParseMediaType(fCt); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	} else {
		mediaType := map[string]bool{
			"image/png":  true,
			"image/jpeg": true,
		}
		if !mediaType[mt] {
			respondWithError(w, http.StatusBadRequest, "bad file format", fmt.Errorf("bad file format"))
			return
		}

		video, err := cfg.db.GetVideo(videoID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error(), err)
			return
		}
		if video.UserID != userID {
			respondWithError(w, http.StatusUnauthorized, "user unauthorized", err)
			return
		}

		ext := strings.Split(fCt, "/")[1]
		fp := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoID.String(), ext))
		f, err := os.Create(fp)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error(), err)
			return
		}
		if _, err := io.Copy(f, file); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error(), err)
			return
		}

		dUrl := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoID.String(), ext)
		video.ThumbnailURL = &dUrl
		if err := cfg.db.UpdateVideo(video); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error(), err)
			return
		}

		respondWithJSON(w, http.StatusOK, video)
	}
}
