package sync

import (
	"bytes"
	"encoding/json"
	"fmt"

	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zionwu/catalog-images-synchronizer/config"
)

type Synchronizer interface {
	Run() error
}

type imageSynchronizer struct {
}

func NewImageSynchronize() Synchronizer {
	return &imageSynchronizer{}
}

func (s *imageSynchronizer) Run() error {

	catalogPath, err := getCatalogFromGitRepos()
	defer os.RemoveAll(catalogPath)
	if err != nil {
		return err
	}

	catalogImages, err := getImagesFromDockerCompose(catalogPath)
	if err != nil {
		return err
	}

	err = synchronizeImages(catalogImages)
	if err != nil {
		return err
	}

	return nil
}

func synchronizeImages(catalogImages map[string][]string) error {
	for catalog, images := range catalogImages {

		logrus.Infof("Start sychronizing catalog %s.....", catalog)

		for _, image := range images {

			logrus.Infof("Start sychronizing image %s.....", image)

			if err := pullImageFromDockerHub(image); err != nil {
				logrus.Errorf("Error occurred while pulling image from dockerhub %v", err)
				continue
			}

			if err := pushImage2Harbor(image); err != nil {
				logrus.Errorf("Error occurred while pushing image to harbor %v", err)
				continue
			}
		}
	}

	return nil

}

func pushImage2Harbor(image string) error {

	c := config.GetConfig()

	cmd := exec.Command("docker", "tag", image, c.HarborAddress+"/"+image)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return err
	}

	parts := strings.Split(image, "/")
	var repo string
	if len(parts) == 1 {
		repo = "library"
	} else {
		repo = parts[0]
	}

	logrus.Infof("Pushing image %s to harbor.....", image)
	exist, err := checkHarborRepoExist(repo)
	if err != nil {
		return err
	}
	if !exist {
		if err := createHarborRepo(repo); err != nil {
			return err
		}
	}

	cmdPush := exec.Command("docker", "push", c.HarborAddress+"/"+image)
	cmdPush.Stderr = os.Stderr
	cmdPush.Stdout = os.Stdout
	if err := cmdPush.Run(); err != nil {
		return err
	}

	return nil
}

func createHarborRepo(repo string) error {

	logrus.Infof("Creating harbor repo %s .....", repo)

	c := config.GetConfig()

	project := struct {
		ProjectName string `json:"project_name"`
		Public      int    `json:"public"`
	}{
		ProjectName: repo,
		Public:      1,
	}

	projectData, err := json.Marshal(project)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+c.HarborAddress+"/api/projects", bytes.NewBuffer(projectData))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.HarborUserName, c.HarborPassword)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	requestBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	logrus.Debug(string(requestBytes))

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Error creating repos %s, code %s", repo, resp.StatusCode)
	}

	logrus.Infof("Created harbor repo %s successfully", repo)

	return nil
}

func checkHarborRepoExist(repo string) (bool, error) {
	c := config.GetConfig()

	req, err := http.NewRequest(http.MethodHead, "http://"+c.HarborAddress+"/api/projects/?project_name="+repo, nil)
	if err != nil {
		return false, err
	}

	req.SetBasicAuth(c.HarborUserName, c.HarborPassword)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	requestBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	logrus.Debug(string(requestBytes))

	return true, nil

}

func pullImageFromDockerHub(image string) error {
	cmd := exec.Command("docker", "pull", image)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func getImagesFromDockerCompose(catalogPath string) (map[string][]string, error) {
	catalogImages := map[string][]string{}

	if err := filepath.Walk(catalogPath, func(fullPath string, f os.FileInfo, err error) error {
		if f == nil || !f.Mode().IsRegular() {
			return nil
		}

		relativePath, err := filepath.Rel(catalogPath, fullPath)
		if err != nil {
			return err
		}
		_, filename := path.Split(relativePath)
		if filename == "docker-compose.yml" || filename == "docker-compose.yml.tpl" {
			dirs := strings.Split(relativePath, "/")
			catalogName := dirs[1]

			composeFile, err := ioutil.ReadFile(fullPath)
			if err != nil {
				return err
			}

			r, _ := regexp.Compile("\\s+image\\s*:(.*?)\n")
			images := r.FindAllString(string(composeFile), -1)

			imageList := catalogImages[catalogName]
			if imageList == nil {
				imageList = []string{}
			}

			for _, imageStr := range images {
				image := imageStr[strings.Index(imageStr, ":")+1:]
				image = strings.TrimSpace(image)

				imageList = append(imageList, image)
			}
			catalogImages[catalogName] = imageList
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return catalogImages, nil
}

func getCatalogFromGitRepos() (string, error) {
	c := config.GetConfig()

	logrus.Infof("Cloning catalog %s", c.CatalogUrl)
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	repoPath, err := ioutil.TempDir(wd, "rancher-catalog-")
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	if c.CatalogBranch == "" {
		cmd = exec.Command("git", "clone", c.CatalogUrl, repoPath)
	} else {
		cmd = exec.Command("git", "clone", "-b", c.CatalogBranch, "--single-branch", c.CatalogBranch, repoPath)
	}

	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return repoPath, nil

}
