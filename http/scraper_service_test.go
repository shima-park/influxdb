package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/inmem"
	"github.com/influxdata/influxdb/mock"
	platformtesting "github.com/influxdata/influxdb/testing"
	"github.com/julienschmidt/httprouter"
)

const (
	targetOneIDString = "0000000000000111"
	targetTwoIDString = "0000000000000222"
)

var (
	targetOneID = platformtesting.MustIDBase16(targetOneIDString)
	targetTwoID = platformtesting.MustIDBase16(targetTwoIDString)
)

func TestService_handleGetScraperTargets(t *testing.T) {
	type fields struct {
		Service platform.ScraperTargetStoreService
	}

	type args struct {
		queryParams map[string][]string
	}

	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get all scraper targets",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					ListTargetsF: func(ctx context.Context) ([]platform.ScraperTarget, error) {
						return []platform.ScraperTarget{
							{
								ID:         targetOneID,
								Name:       "target-1",
								Type:       platform.PrometheusScraperType,
								URL:        "www.one.url",
								OrgName:    "org-name",
								BucketName: "bkt-one",
							},
							{
								ID:         targetTwoID,
								Name:       "target-2",
								Type:       platform.PrometheusScraperType,
								URL:        "www.two.url",
								OrgName:    "org-name",
								BucketName: "bkt-two",
							},
						}, nil
					},
				},
			},
			args: args{},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: fmt.Sprintf(
					`
					{
					  "links": {
					    "self": "/api/v2/scrapertargets"
					  },
					  "scraper_targets": [
					    {
					      "id": "%s",
						  "name": "target-1",
						  "bucket": "bkt-one",
						  "org": "org-name",
						  "type": "prometheus",
						  "url": "www.one.url",
						  "links": {
						    "self": "/api/v2/scrapertargets/0000000000000111"
						  }
						},
						{
						  "id": "%s",
						  "name": "target-2",
						  "bucket": "bkt-two",
						  "org": "org-name",
						  "type": "prometheus",
						  "url": "www.two.url",
						  "links": {
						    "self": "/api/v2/scrapertargets/0000000000000222"
						  }
                        }
					  ]
					}
					`,
					targetOneIDString,
					targetTwoIDString,
				),
			},
		},
		{
			name: "get all scraper targets when there are none",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					ListTargetsF: func(ctx context.Context) ([]platform.ScraperTarget, error) {
						return []platform.ScraperTarget{}, nil
					},
				},
			},
			args: args{},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
                {
                  "links": {
                    "self": "/api/v2/scrapertargets"
                  },
                  "scraper_targets": []
                }
                `,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewScraperHandler()
			h.ScraperStorageService = tt.fields.Service

			r := httptest.NewRequest("GET", "http://any.tld", nil)

			qp := r.URL.Query()
			for k, vs := range tt.args.queryParams {
				for _, v := range vs {
					qp.Add(k, v)
				}
			}
			r.URL.RawQuery = qp.Encode()

			w := httptest.NewRecorder()

			h.handleGetScraperTargets(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetScraperTargets() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetScraperTargets() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetScraperTargets() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handleGetScraperTarget(t *testing.T) {
	type fields struct {
		Service platform.ScraperTargetStoreService
	}

	type args struct {
		id string
	}

	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get a scraper target by id",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					GetTargetByIDF: func(ctx context.Context, id platform.ID) (*platform.ScraperTarget, error) {
						if id == targetOneID {
							return &platform.ScraperTarget{
								ID:         targetOneID,
								Name:       "target-1",
								Type:       platform.PrometheusScraperType,
								URL:        "www.some.url",
								OrgName:    "org-name",
								BucketName: "bkt-name",
							}, nil
						}
						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: targetOneIDString,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: fmt.Sprintf(
					`
                    {
                      "id": "%[1]s",
                      "name": "target-1",
                      "type": "prometheus",
                      "url": "www.some.url",
                      "bucket": "bkt-name",
                      "org": "org-name",
                      "links": {
                        "self": "/api/v2/scrapertargets/%[1]s"
                      }
                    }
                    `,
					targetOneIDString,
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewScraperHandler()
			h.ScraperStorageService = tt.fields.Service

			r := httptest.NewRequest("GET", "http://any.tld", nil)

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleGetScraperTarget(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetScraperTarget() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetScraperTarget() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetScraperTarget() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handleDeleteScraperTarget(t *testing.T) {
	type fields struct {
		Service platform.ScraperTargetStoreService
	}

	type args struct {
		id string
	}

	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "delete a scraper target by id",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					RemoveTargetF: func(ctx context.Context, id platform.ID) error {
						if id == targetOneID {
							return nil
						}

						return fmt.Errorf("wrong id")
					},
				},
			},
			args: args{
				id: targetOneIDString,
			},
			wants: wants{
				statusCode: http.StatusNoContent,
			},
		},
		{
			name: "scraper target not found",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					RemoveTargetF: func(ctx context.Context, id platform.ID) error {
						return &platform.Error{
							Code: platform.ENotFound,
							Msg:  platform.ErrScraperTargetNotFound,
						}
					},
				},
			},
			args: args{
				id: targetTwoIDString,
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewScraperHandler()
			h.ScraperStorageService = tt.fields.Service

			r := httptest.NewRequest("GET", "http://any.tld", nil)

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleDeleteScraperTarget(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleDeleteScraperTarget() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleDeleteScraperTarget() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleDeleteScraperTarget() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handlePostScraperTarget(t *testing.T) {
	type fields struct {
		Service platform.ScraperTargetStoreService
	}

	type args struct {
		target *platform.ScraperTarget
	}

	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "create a new scraper target",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					AddTargetF: func(ctx context.Context, st *platform.ScraperTarget) error {
						st.ID = targetOneID
						return nil
					},
				},
			},
			args: args{
				target: &platform.ScraperTarget{
					Name:       "hello",
					Type:       platform.PrometheusScraperType,
					BucketName: "bkt-name",
					OrgName:    "org-name",
					URL:        "www.some.url",
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: fmt.Sprintf(
					`
                    {
                      "id": "%[1]s",
                      "name": "hello",
                      "type": "prometheus",
                      "url": "www.some.url",
                      "org": "org-name",
                      "bucket": "bkt-name",
                      "links": {
                        "self": "/api/v2/scrapertargets/%[1]s"
                      }
                    }
                    `,
					targetOneIDString,
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewScraperHandler()
			h.ScraperStorageService = tt.fields.Service

			st, err := json.Marshal(tt.args.target)
			if err != nil {
				t.Fatalf("failed to unmarshal scraper target: %v", err)
			}

			r := httptest.NewRequest("GET", "http://any.tld", bytes.NewReader(st))
			w := httptest.NewRecorder()

			h.handlePostScraperTarget(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostScraperTarget() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostScraperTarget() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePostScraperTarget() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handlePatchScraperTarget(t *testing.T) {
	type fields struct {
		Service platform.ScraperTargetStoreService
	}

	type args struct {
		id     string
		update *platform.ScraperTarget
	}

	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "update a scraper target",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					UpdateTargetF: func(ctx context.Context, t *platform.ScraperTarget) (*platform.ScraperTarget, error) {
						if t.ID == targetOneID {
							return t, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: targetOneIDString,
				update: &platform.ScraperTarget{
					ID:         targetOneID,
					Name:       "name",
					BucketName: "buck",
					Type:       platform.PrometheusScraperType,
					URL:        "www.example.url",
					OrgName:    "orgg",
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: fmt.Sprintf(
					`
		            {
		              "id":"%[1]s",
		              "name":"name",
		              "type":"prometheus",
		              "url":"www.example.url",
		              "org":"orgg",
		              "bucket":"buck",
		              "links":{
		                "self":"/api/v2/scrapertargets/%[1]s"
		              }
		            }
		            `,
					targetOneIDString,
				),
			},
		},
		{
			name: "scraper target not found",
			fields: fields{
				Service: &mock.ScraperTargetStoreService{
					UpdateTargetF: func(ctx context.Context, upd *platform.ScraperTarget) (*platform.ScraperTarget, error) {
						return nil, &platform.Error{
							Code: platform.ENotFound,
							Msg:  platform.ErrScraperTargetNotFound,
						}
					},
				},
			},
			args: args{
				id: targetOneIDString,
				update: &platform.ScraperTarget{
					ID:         targetOneID,
					Name:       "name",
					BucketName: "buck",
					Type:       platform.PrometheusScraperType,
					URL:        "www.example.url",
					OrgName:    "orgg",
				},
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewScraperHandler()
			h.ScraperStorageService = tt.fields.Service

			var err error
			st := make([]byte, 0)
			if tt.args.update != nil {
				st, err = json.Marshal(*tt.args.update)
				if err != nil {
					t.Fatalf("failed to unmarshal scraper target: %v", err)
				}
			}

			fmt.Println(string(st))

			r := httptest.NewRequest("GET", "http://any.tld", bytes.NewReader(st))

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handlePatchScraperTarget(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePatchScraperTarget() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePatchScraperTarget() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePatchScraperTarget() = ***%s***", tt.name, diff)
			}
		})
	}
}

func initScraperService(f platformtesting.TargetFields, t *testing.T) (platform.ScraperTargetStoreService, string, func()) {
	t.Helper()
	svc := inmem.NewService()
	svc.IDGenerator = f.IDGenerator

	ctx := context.Background()
	for _, target := range f.Targets {
		if err := svc.PutTarget(ctx, target); err != nil {
			t.Fatalf("failed to populate scraper targets")
		}
	}

	handler := NewScraperHandler()
	handler.ScraperStorageService = svc
	server := httptest.NewServer(handler)
	client := ScraperService{
		Addr:     server.URL,
		OpPrefix: inmem.OpPrefix,
	}
	done := server.Close

	return &client, inmem.OpPrefix, done
}

func TestScraperService(t *testing.T) {
	platformtesting.ScraperService(initScraperService, t)
}
