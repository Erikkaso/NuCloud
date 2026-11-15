package main

import (
  "os"
  "log"
  "github.com/gin-gonic/gin"
  "github.com/joho/godotenv"
  "gitlab.com/gitlab-org/api/client-go/v3"
)

func buildapp(c *gin.Context) {
  git, err := gitlab.NewClient(os.Getenv("GITLAB_CLIENT_TOKEN"))
	if err != nil {
		log.Printf("Failed to create client: %v", err)
    c.JSON(500, gin.H{"warning": "INTERNAL ERROR"})
	}
  
  url := c.PostForm("url")
  
  projectID := os.Getenv("PROJECT_ID")
  opts := &gitlab.RunPipelineTriggerOptions{
      Ref: gitlab.Ptr("main"),
      Token: gitlab.Ptr(os.Getenv("GITLAB_PIPELINE_TOKEN")),
      Variables: map[string]string{
        "REPO_URL": url,
        },
      }

  _, _, err = git.PipelineTriggers.RunPipelineTrigger(projectID, opts)
	if err != nil {
		log.Printf("Erro ao acionar o pipeline: %v", err)
    c.JSON(500, gin.H{"warning": "INTERNAL ERROR"})
	}

  c.JSON(200, gin.H{"warning":"operação bem sucedida"})

}

func main() {
    if err := godotenv.Load(); err != nil {
      log.Printf("Erro ao carregar .env: %v", err)
  }
  router := gin.Default()
  router.POST("/buildapp", buildapp)
  router.Run()
}

