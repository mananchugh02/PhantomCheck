package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	gh "github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

type Client struct{ *gh.Client }

func NewClient(token string) *Client {
	if token == "" {
		return &Client{Client: gh.NewClient(nil)}
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return &Client{Client: gh.NewClient(oauth2.NewClient(context.Background(), ts))}
}

func GetChangedGoFiles(ctx context.Context, client *Client, owner, repo string, prNumber int) ([]string, error) {
	var files []string
	opts := &gh.ListOptions{PerPage: 100}
	for {
		chunk, resp, err := client.PullRequests.ListFiles(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, err
		}
		for _, file := range chunk {
			if strings.HasSuffix(strings.ToLower(file.GetFilename()), ".go") {
				files = append(files, file.GetFilename())
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return files, nil
		}
		opts.Page = resp.NextPage
	}
}

func GetFileContent(ctx context.Context, client *Client, owner, repo, filepath, ref string) (string, error) {
	content, _, _, err := client.Repositories.GetContents(ctx, owner, repo, filepath, &gh.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return "", err
	}
	if content == nil {
		return "", fmt.Errorf("no content returned for %s", filepath)
	}
	decoded, err := content.GetContent()
	if err != nil {
		if content.GetDownloadURL() == "" {
			return "", err
		}
		return fetchDownload(ctx, content.GetDownloadURL())
	}
	return decoded, nil
}

func GetPRHeadSHA(ctx context.Context, client *Client, owner, repo string, prNumber int) (string, error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return "", err
	}
	if pr.GetHead() == nil {
		return "", fmt.Errorf("pull request head is missing")
	}
	return pr.GetHead().GetSHA(), nil
}

func fetchDownload(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
