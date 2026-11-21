package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Source struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note"`
}

type Skill struct {
	ID          string   `json:"id"`
	Source      string   `json:"source"`
	SourceName  string   `json:"sourceName"`
	Slug        string   `json:"slug"`
	Namespace   string   `json:"namespace"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Installs    int      `json:"installs"`
	Tags        []string `json:"tags"`
	Category    string   `json:"category"`
	URL         string   `json:"url"`
	Repository  string   `json:"repository"`
	IconURL     string   `json:"iconUrl"`
	Installed   bool     `json:"installed"`
}

type Result struct {
	Items []Skill `json:"items"`
	Total int     `json:"total"`
	Page  int     `json:"page"`
}

type Provider interface {
	ID() string
	Search(ctx context.Context, query, category string, page int) ([]Skill, int, error)
	Download(ctx context.Context, skill Skill, destination string) error
}

func Sources() []Source {
	return []Source{
		{ID: "skillhub", Name: "SkillHub", Status: "available", Note: "中文社区公开 API"},
		{ID: "skills-sh", Name: "skills.sh", Status: "available", Note: "sitemap 目录 + GitHub 仓库安装"},
		{ID: "skillsmp", Name: "SkillsMP", Status: "experimental", Note: "Cloudflare 防护，暂未启用"},
	}
}

func DefaultProviders() map[string]Provider {
	return map[string]Provider{
		"skillhub":  NewSkillHub(),
		"skills-sh": NewSkillsSh(),
	}
}

type SkillHub struct {
	client *http.Client
}

func NewSkillHub() *SkillHub {
	return &SkillHub{client: &http.Client{Timeout: 25 * time.Second}}
}

func (p *SkillHub) ID() string { return "skillhub" }

type skillHubResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Skills []struct {
			Slug        string `json:"slug"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Category    string `json:"category"`
			Version     string `json:"version"`
			Installs    int    `json:"installs"`
			Downloads   int    `json:"downloads"`
			IconURL     string `json:"iconUrl"`
			Homepage    string `json:"homepage"`
			OwnerName   string `json:"ownerName"`
			Namespace   struct {
				Handle string `json:"handle"`
			} `json:"namespace"`
		} `json:"skills"`
		Total int `json:"total"`
	} `json:"data"`
}

func (p *SkillHub) Search(ctx context.Context, query, category string, page int) ([]Skill, int, error) {
	if page < 1 {
		page = 1
	}
	values := url.Values{}
	values.Set("page", strconv.Itoa(page))
	values.Set("pageSize", "24")
	if strings.TrimSpace(query) != "" {
		values.Set("keyword", strings.TrimSpace(query))
	}
	if strings.TrimSpace(category) != "" {
		values.Set("category", strings.TrimSpace(category))
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.skillhub.cn/api/skills?"+values.Encode(), nil)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("User-Agent", "XinSkillVault/0.1")
	request.Header.Set("Referer", "https://www.skillhub.cn/")
	response, err := p.client.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("skillhub 请求失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("skillhub 返回 %d", response.StatusCode)
	}
	var payload skillHubResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, 0, fmt.Errorf("解析 skillhub 响应: %w", err)
	}
	if payload.Code != 0 {
		return nil, 0, fmt.Errorf("skillhub: %s", payload.Message)
	}
	items := make([]Skill, 0, len(payload.Data.Skills))
	for _, item := range payload.Data.Skills {
		namespace := item.Namespace.Handle
		installURL := "https://www.skillhub.cn/" + namespace + "/" + item.Slug
		items = append(items, Skill{
			ID: "skillhub:" + namespace + ":" + item.Slug, Source: "skillhub", SourceName: "SkillHub",
			Slug: item.Slug, Namespace: namespace, Name: item.Name, Description: item.Description,
			Author: item.OwnerName, Version: item.Version, Installs: item.Installs,
			Category: item.Category, Tags: []string{}, URL: installURL, IconURL: item.IconURL,
		})
	}
	return items, payload.Data.Total, nil
}

func (p *SkillHub) Download(ctx context.Context, skill Skill, destination string) error {
	if skill.Source != "skillhub" || skill.Slug == "" {
		return fmt.Errorf("不支持的技能来源")
	}
	values := url.Values{}
	values.Set("slug", skill.Slug)
	if skill.Namespace != "" {
		values.Set("namespace", skill.Namespace)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.skillhub.cn/api/v1/download?"+values.Encode(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "XinSkillVault/0.1")
	request.Header.Set("Referer", "https://www.skillhub.cn/")
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("下载技能失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("下载技能返回 %d", response.StatusCode)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, io.LimitReader(response.Body, 200<<20)); err != nil {
		file.Close()
		return fmt.Errorf("写入技能包: %w", err)
	}
	return file.Close()
}

type Category struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}
