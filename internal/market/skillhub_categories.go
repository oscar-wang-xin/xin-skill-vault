package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type skillHubCategories struct {
	Items []struct {
		Key    string `json:"key"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
	} `json:"items"`
}

func (p *SkillHub) Categories(ctx context.Context) ([]Category, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.skillhub.cn/api/v1/categories", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "XinSkillVault/0.1")
	request.Header.Set("Referer", "https://www.skillhub.cn/")
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skillhub categories 返回 %d", response.StatusCode)
	}
	var payload skillHubCategories
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]Category, 0, len(payload.Items))
	for _, item := range payload.Items {
		if !item.Active {
			continue
		}
		out = append(out, Category{Key: item.Key, Name: item.Name})
	}
	return out, nil
}
