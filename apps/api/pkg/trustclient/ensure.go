package trustclient

import "context"

type EnsureSubjectKindItem struct {
	Key string `json:"key"`
}

type EnsureSubjectKindResult struct {
	Key    string `json:"key"`
	Result string `json:"result"`
}

func (c *Client) EnsureSubjectKinds(
	ctx context.Context, kinds []EnsureSubjectKindItem,
) ([]EnsureSubjectKindResult, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	body := struct {
		Kinds []EnsureSubjectKindItem `json:"kinds"`
	}{Kinds: kinds}
	var out struct {
		Results []EnsureSubjectKindResult `json:"results"`
	}
	if err := c.postJSON(ctx, "/api/v1/trust/subject-kinds/ensure", body, &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}
