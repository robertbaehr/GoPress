package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type postWriteBody struct {
	Title   json.RawMessage `json:"title"`
	Content json.RawMessage `json:"content"`
	Excerpt json.RawMessage `json:"excerpt"`
	Status  string          `json:"status"`
	Slug    string          `json:"slug"`
	Author  *int64          `json:"author"`
	// Sticky is optional; nil means leave sticky_posts unchanged (P4.97).
	Sticky *bool `json:"sticky"`
	// MenuOrder is optional page-attributes order (P4.157).
	MenuOrder *int `json:"menu_order"`
	// Meta is optional registered post meta (footnotes); nil means omit (P4.130).
	Meta map[string]any `json:"meta"`
}

func (s *Server) servePostCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	switch r.Method {
	case http.MethodGet:
		s.listPostCollection(w, r, c, postType)
	case http.MethodPost:
		// P4.150: create for post and page; type is forced from the route.
		if postType != "post" && postType != "page" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 405},
			})
			return
		}
		s.createTypedPost(w, r, c, postType)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) listPostCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	// P4.138: strict per_page validation (1–100) like rest_validate_request_arg.
	perPage, perOK := parsePostsPerPage(r, 10)
	if !perOK {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): per_page",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{
					"per_page": "per_page must be between 1 (inclusive) and 100 (inclusive)",
				},
			},
		})
		return
	}
	page := parsePage(r)
	// P4.139: strict offset integer validation like rest_validate_request_arg.
	offset, useOffset, offsetOK := parsePostsOffset(r)
	if !offsetOK {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): offset",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{
					"offset": "offset is not of type integer.",
				},
			},
		})
		return
	}
	// P4.145: context enum + edit-context capability gate.
	restCtx, ctxOK := parsePostsRESTContext(r)
	if !ctxOK {
		writeRESTInvalidPostsContext(w)
		return
	}
	if restCtx == "edit" && !canEditPosts(c) {
		writeRESTForbiddenPostsContext(w, c, false)
		return
	}
	edit := restCtx == "edit"
	// P4.141 / P4.162: sticky / ignore_sticky only registered for post (WP schema).
	// Other types ignore the params (additional properties not validated).
	var stickyFilter *bool
	ignoreSticky := true
	if postType == "post" {
		var stickyOK, ignoreOK bool
		stickyFilter, stickyOK = parseStickyQuery(r)
		ignoreSticky, ignoreOK = parseIgnoreStickyQuery(r)
		var badBools []string
		if !stickyOK {
			badBools = append(badBools, "sticky")
		}
		if !ignoreOK {
			badBools = append(badBools, "ignore_sticky")
		}
		if len(badBools) > 0 {
			writeRESTInvalidBooleanParams(w, badBools)
			return
		}
	}
	// P4.149 / P4.159: strict order / orderby enum + string-type validation.
	orderby, orderbyOK, orderbyTypeErr := parsePostsOrderbyTyped(r, postType)
	order, orderOK, orderTypeErr := parsePostsOrderTyped(r)
	var badStringTypes []string
	if orderbyTypeErr {
		badStringTypes = append(badStringTypes, "orderby")
	}
	if orderTypeErr {
		badStringTypes = append(badStringTypes, "order")
	}
	if len(badStringTypes) > 0 {
		writeRESTInvalidStringParams(w, badStringTypes)
		return
	}
	badOrder := map[string]string{}
	if !orderOK {
		badOrder["order"] = "order is not one of asc and desc."
	}
	if !orderbyOK {
		badOrder["orderby"] = postsOrderbyEnumMessage(postType)
	}
	if len(badOrder) > 0 {
		writeRESTInvalidOrderParams(w, badOrder)
		return
	}
	search := parsePostsSearch(r)
	// P4.153: strict search_columns enum validation (rest_not_in_enum).
	searchColumns, searchColumnsErr := parsePostsSearchColumns(r)
	if searchColumnsErr != "" {
		writeRESTInvalidOrderParams(w, map[string]string{"search_columns": searchColumnsErr})
		return
	}
	authorIn := parsePostsAuthor(r)
	// P4.152: strict author_exclude integer-array validation (rest_invalid_type).
	authorNotIn, authorExOK := parsePostsAuthorExclude(r)
	if !authorExOK {
		writeRESTInvalidIntegerArrayParams(w, []string{"author_exclude"})
		return
	}
	categories, categoryOperator := parsePostsCategories(r, c)
	// P4.155: strict categories_exclude integer-array validation (rest_invalid_type).
	// P4.161: object form include_children expands hierarchical descendants.
	categoriesExclude, catExOK := parsePostsCategoriesExclude(r, c)
	if !catExOK {
		writeRESTInvalidIntegerArrayParams(w, []string{"categories_exclude"})
		return
	}
	tags, tagOperator := parsePostsTags(r)
	// P4.156: strict tags_exclude integer-array validation (rest_invalid_type).
	tagsExclude, tagsExOK := parsePostsTagsExclude(r)
	if !tagsExOK {
		writeRESTInvalidIntegerArrayParams(w, []string{"tags_exclude"})
		return
	}
	// P4.164: strict tax_relation enum validation (rest_not_in_enum).
	taxRelation, taxRelationErr := parsePostsTaxRelation(r)
	if taxRelationErr != "" {
		writeRESTInvalidOrderParams(w, map[string]string{"tax_relation": taxRelationErr})
		return
	}
	slugs := parsePostsSlug(r)
	include := parsePostsInclude(r)
	exclude := parsePostsExclude(r)
	// P4.131: WP REST validates orderby=relevance / orderby=include prerequisites.
	if orderby == "relevance" && search == "" {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_no_search_term_defined",
			"message": "You need to define a search term to order by relevance.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	if orderby == "include" && len(include) == 0 {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_orderby_include_missing_include",
			"message": "You need to define an include parameter to order by include.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	// P4.96 / P4.154: format= post_format filter (posts only; strict enum).
	var formatIn []string
	var formatStandard bool
	if postType == "post" {
		var formatErr string
		formatIn, formatStandard, formatErr = parsePostsFormat(r)
		if formatErr != "" {
			writeRESTInvalidOrderParams(w, map[string]string{"format": formatErr})
			return
		}
	}
	// P4.93 / P4.142: parent / parent_exclude for hierarchical types (pages).
	// P4.149: menu_order equality filter (pages / page-attributes only).
	var parentIn, parentNotIn []int64
	var menuOrderFilter *int
	if postType == "page" {
		var parentOK, parentExOK bool
		parentIn, parentOK = parsePostsParent(r)
		parentNotIn, parentExOK = parsePostsParentExclude(r)
		var badParents []string
		if !parentOK {
			badParents = append(badParents, "parent")
		}
		if !parentExOK {
			badParents = append(badParents, "parent_exclude")
		}
		if len(badParents) > 0 {
			writeRESTInvalidIntegerArrayParams(w, badParents)
			return
		}
		mo, useMO, moOK := parsePostsMenuOrder(r)
		if !moOK {
			w.Header().Set("Allow", "GET")
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): menu_order",
				"data": map[string]any{
					"status": 400,
					"params": map[string]any{
						"menu_order": "menu_order is not of type integer.",
					},
					"details": map[string]any{
						"menu_order": map[string]any{
							"code":    "rest_invalid_type",
							"message": "menu_order is not of type integer.",
							"data":    map[string]any{"param": "menu_order"},
						},
					},
				},
			})
			return
		}
		if useMO {
			menuOrderFilter = &mo
		}
	}
	// P4.140 / P4.165: strict date-time validation + string-type for after/before/modified_*.
	dateAfter, afterOK, afterTypeErr := parsePostsAfterTyped(r)
	dateBefore, beforeOK, beforeTypeErr := parsePostsBeforeTyped(r)
	modifiedAfter, modAfterOK, modAfterTypeErr := parsePostsModifiedAfterTyped(r)
	modifiedBefore, modBeforeOK, modBeforeTypeErr := parsePostsModifiedBeforeTyped(r)
	var badDateTypes []string
	// Schema order from WP_REST_Posts_Controller::get_collection_params.
	if afterTypeErr {
		badDateTypes = append(badDateTypes, "after")
	}
	if modAfterTypeErr {
		badDateTypes = append(badDateTypes, "modified_after")
	}
	if beforeTypeErr {
		badDateTypes = append(badDateTypes, "before")
	}
	if modBeforeTypeErr {
		badDateTypes = append(badDateTypes, "modified_before")
	}
	if len(badDateTypes) > 0 {
		writeRESTInvalidStringParams(w, badDateTypes)
		return
	}
	var badDates []string
	if !afterOK {
		badDates = append(badDates, "after")
	}
	if !modAfterOK {
		badDates = append(badDates, "modified_after")
	}
	if !beforeOK {
		badDates = append(badDates, "before")
	}
	if !modBeforeOK {
		badDates = append(badDates, "modified_before")
	}
	if len(badDates) > 0 {
		writeRESTInvalidDateParams(w, badDates)
		return
	}
	// P4.166/P4.170/P4.172: meta_key/meta_value/meta_compare/meta_type/meta_compare_key/meta_type_key string-type; meta_query object rejected.
	if parsePostsMetaQueryReject(r) {
		writeRESTInvalidStringParams(w, []string{"meta_query"})
		return
	}
	metaKey, _, metaKeyTypeErr := parsePostsMetaKeyTyped(r)
	metaValue, _, metaValueTypeErr := parsePostsMetaValueTyped(r)
	metaCompare, metaCompareOK, metaCompareTypeErr, metaCompareEnumErr := parsePostsMetaCompareTyped(r)
	metaType, metaTypeOK, metaTypeTypeErr, metaTypeEnumErr := parsePostsMetaTypeTyped(r)
	metaCompareKey, metaCompareKeyOK, metaCompareKeyTypeErr, metaCompareKeyEnumErr := parsePostsMetaCompareKeyTyped(r)
	metaTypeKey, metaTypeKeyOK, metaTypeKeyTypeErr, metaTypeKeyEnumErr := parsePostsMetaTypeKeyTyped(r)
	var badMetaTypes []string
	if metaKeyTypeErr {
		badMetaTypes = append(badMetaTypes, "meta_key")
	}
	if metaValueTypeErr {
		badMetaTypes = append(badMetaTypes, "meta_value")
	}
	if metaCompareTypeErr {
		badMetaTypes = append(badMetaTypes, "meta_compare")
	}
	if metaTypeTypeErr {
		badMetaTypes = append(badMetaTypes, "meta_type")
	}
	if metaCompareKeyTypeErr {
		badMetaTypes = append(badMetaTypes, "meta_compare_key")
	}
	if metaTypeKeyTypeErr {
		badMetaTypes = append(badMetaTypes, "meta_type_key")
	}
	if len(badMetaTypes) > 0 {
		writeRESTInvalidStringParams(w, badMetaTypes)
		return
	}
	if metaCompareEnumErr != "" || !metaCompareOK {
		msg := metaCompareEnumErr
		if msg == "" {
			msg = postsMetaCompareEnumMessage()
		}
		writeRESTInvalidOrderParams(w, map[string]string{"meta_compare": msg})
		return
	}
	if metaTypeEnumErr != "" || !metaTypeOK {
		msg := metaTypeEnumErr
		if msg == "" {
			msg = postsMetaTypeEnumMessage()
		}
		writeRESTInvalidOrderParams(w, map[string]string{"meta_type": msg})
		return
	}
	if metaCompareKeyEnumErr != "" || !metaCompareKeyOK {
		msg := metaCompareKeyEnumErr
		if msg == "" {
			msg = postsMetaCompareKeyEnumMessage()
		}
		writeRESTInvalidOrderParams(w, map[string]string{"meta_compare_key": msg})
		return
	}
	if metaTypeKeyEnumErr != "" || !metaTypeKeyOK {
		msg := metaTypeKeyEnumErr
		if msg == "" {
			msg = postsMetaTypeKeyEnumMessage()
		}
		writeRESTInvalidOrderParams(w, map[string]string{"meta_type_key": msg})
		return
	}
	statuses, forbiddenCode, enumInvalid := parsePostsStatus(r, c)
	if forbiddenCode != 0 || enumInvalid {
		// P4.151: WP wraps status sanitize errors as rest_invalid_param.
		writeRESTInvalidPostsStatus(w, forbiddenCode, enumInvalid)
		return
	}
	total, err := countPosts(c, postType, stickyFilter, search, searchColumns, authorIn, authorNotIn, categories, categoriesExclude, tags, tagsExclude, taxRelation, categoryOperator, tagOperator, statuses, slugs, include, exclude, parentIn, parentNotIn, dateAfter, dateBefore, modifiedAfter, modifiedBefore, formatIn, formatStandard, menuOrderFilter, metaKey, metaValue, metaCompare, metaType, metaCompareKey, metaTypeKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	posts, err := listPostsPaged(c, postType, perPage, page, edit, stickyFilter, ignoreSticky, orderby, order, search, searchColumns, authorIn, authorNotIn, categories, categoriesExclude, tags, tagsExclude, taxRelation, categoryOperator, tagOperator, statuses, slugs, include, exclude, parentIn, parentNotIn, dateAfter, dateBefore, modifiedAfter, modifiedBefore, formatIn, formatStandard, offset, useOffset, menuOrderFilter, metaKey, metaValue, metaCompare, metaType, metaCompareKey, metaTypeKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	if canEditPosts(c) {
		w.Header().Set("Allow", "GET, POST")
	}
	applyPostsRESTContext(posts, restCtx)
	// P4.146: ?_embed → _embedded author / featuredmedia / terms.
	applyPostsRESTEmbed(c, posts, r)
	// P9.26: merge register_rest_field get_callback values when registered.
	// P4.147: ?_fields → sparse field projection (rest_filter_response_fields).
	payload := EncodeWithAdditionalFields(postType, posts, r)
	writeJSON(w, http.StatusOK, applyRESTResponseFields(r, payload))
}

func (s *Server) serveTypedItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string, id int64) {
	switch r.Method {
	case http.MethodGet:
		s.getTypedItem(w, r, c, postType, id)
	case http.MethodPut, http.MethodPost, http.MethodPatch:
		if postType != "post" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 405},
			})
			return
		}
		s.updatePost(w, r, c, id)
	case http.MethodDelete:
		if postType != "post" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 405},
			})
			return
		}
		s.deletePost(w, r, c, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) getTypedItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string, id int64) {
	// P4.145: context enum + edit-context capability gate.
	restCtx, ctxOK := parsePostsRESTContext(r)
	if !ctxOK {
		writeRESTInvalidPostsContext(w)
		return
	}
	if restCtx == "edit" && !canEditPosts(c) {
		writeRESTForbiddenPostsContext(w, c, true)
		return
	}
	edit := restCtx == "edit"
	var (
		post *Post
		err  error
	)
	// Editors may read drafts/private via any-status lookup (P10.11 e2e create→get).
	if edit {
		post, err = getPostByIDAny(c, postType, id, true)
	} else {
		post, err = getPostByID(c, postType, id, false)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if canEditPosts(c) {
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	}
	post.RESTContext = restCtx
	// P4.146: ?_embed → _embedded author / featuredmedia / terms.
	applyPostRESTEmbed(c, post, r)
	// P9.26: merge register_rest_field get_callback values when registered.
	// P4.147: ?_fields → sparse field projection (rest_filter_response_fields).
	payload := EncodeWithAdditionalFields(postType, post, r)
	writeJSON(w, http.StatusOK, applyRESTResponseFields(r, payload))
}

// createTypedPost creates a post-type row; type is forced from the route (P4.150).
// Request body "type" is ignored (WP schema readonly). Replaces createPost (P4.13).
func (s *Server) createTypedPost(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	if postType == "" {
		postType = "post"
	}
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create posts as this user.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "create_posts") && !c.UserCan(c.CurrentUser, "edit_posts") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create posts as this user.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	body, err := decodePostWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	status := body.Status
	if status == "" {
		status = "draft"
	}
	if status == "publish" && !c.UserCan(c.CurrentUser, "publish_posts") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_publish",
			"message": "Sorry, you are not allowed to publish posts.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	title := rawOrRendered(body.Title)
	content := rawOrRendered(body.Content)
	excerpt := rawOrRendered(body.Excerpt)
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		slug = wp.SanitizeTitle(title)
	} else {
		slug = wp.SanitizeTitle(slug)
	}
	if slug == "" {
		slug = postType
	}
	author := c.CurrentUser.ID
	if body.Author != nil && *body.Author > 0 {
		author = *body.Author
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	// Sticky applies to posts only (P4.97 / P4.150).
	if body.Sticky != nil && postType == "post" {
		if code, msg := stickyWriteForbidden(c); code != 0 {
			writeJSON(w, code, map[string]any{
				"code":    "rest_cannot_assign_sticky",
				"message": msg,
				"data":    map[string]any{"status": code},
			})
			return
		}
	}
	menuOrder := 0
	if body.MenuOrder != nil {
		menuOrder = *body.MenuOrder
	}
	id, err := insertPostRow(c, postRowInsert{
		Author:    author,
		Date:      now,
		Content:   content,
		Title:     title,
		Excerpt:   excerpt,
		Status:    status,
		Slug:      slug,
		Type:      postType, // P4.150: route type wins over any body type
		MenuOrder: menuOrder,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if body.Sticky != nil && postType == "post" {
		if err := applyStickyWrite(c, id, *body.Sticky); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_post_sticky",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	}
	if err := applyPostMetaWrite(c, id, body.Meta); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_meta",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	post, err := getPostByIDAny(c, postType, id, true)
	if err != nil || post == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "type": postType})
		return
	}
	// P4.150: defensive — response type always matches the controller.
	post.Type = postType
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, post)
}

func (s *Server) updatePost(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this post.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	existing, err := loadPostRow(c, "post", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_post", id) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	body, err := decodePostWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	title := existing.Title
	content := existing.Content
	excerpt := existing.Excerpt
	slug := existing.Name
	status := existing.Status
	if body.Title != nil {
		title = rawOrRendered(body.Title)
	}
	if body.Content != nil {
		content = rawOrRendered(body.Content)
	}
	if body.Excerpt != nil {
		excerpt = rawOrRendered(body.Excerpt)
	}
	if body.Slug != "" {
		slug = wp.SanitizeTitle(body.Slug)
	}
	if body.Status != "" {
		if body.Status == "publish" && existing.Status != "publish" && !c.UserCan(c.CurrentUser, "publish_posts") {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"code":    "rest_cannot_publish",
				"message": "Sorry, you are not allowed to publish posts.",
				"data":    map[string]any{"status": 403},
			})
			return
		}
		status = body.Status
	}
	if body.Sticky != nil {
		if code, msg := stickyWriteForbidden(c); code != 0 {
			writeJSON(w, code, map[string]any{
				"code":    "rest_cannot_assign_sticky",
				"message": msg,
				"data":    map[string]any{"status": code},
			})
			return
		}
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	changed := title != existing.Title || content != existing.Content || excerpt != existing.Excerpt
	if err := updatePostRow(c, id, title, content, excerpt, slug, status, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_update",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	// P4.157: menu_order write (page-attributes); omit-safe when nil.
	if body.MenuOrder != nil {
		if err := updatePostMenuOrder(c, id, *body.MenuOrder, now); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_post_update",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	}
	// Create a revision of the updated post when revisioned fields change (P4.48).
	if changed && status != "auto-draft" {
		_, _ = savePostRevision(c, existing.Author, id, title, content, excerpt, now)
	}
	if body.Sticky != nil {
		if err := applyStickyWrite(c, id, *body.Sticky); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_post_sticky",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	}
	if err := applyPostMetaWrite(c, id, body.Meta); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_meta",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	post, err := getPostByIDAny(c, "post", id, true)
	if err != nil || post == nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, post)
}

func (s *Server) deletePost(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete this post.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	existing, err := loadPostRow(c, "post", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "delete_post", id) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if err := updatePostStatus(c, id, "trash", now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_delete",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	post, err := getPostByIDAny(c, "post", id, true)
	if err != nil || post == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted":  true,
			"previous": map[string]any{"id": id, "status": existing.Status},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":  true,
		"previous": post,
	})
}

// stickyWriteForbidden mirrors REST sticky capability checks (P4.97).
func stickyWriteForbidden(c *wp.Context) (int, string) {
	if c == nil || c.CurrentUser == nil {
		return http.StatusUnauthorized, "Sorry, you are not allowed to make posts sticky."
	}
	if !c.UserCan(c.CurrentUser, "edit_others_posts") || !c.UserCan(c.CurrentUser, "publish_posts") {
		return http.StatusForbidden, "Sorry, you are not allowed to make posts sticky."
	}
	return 0, ""
}

func applyStickyWrite(c *wp.Context, id int64, sticky bool) error {
	if sticky {
		return wp.StickPost(c, id)
	}
	return wp.UnstickPost(c, id)
}

func decodePostWriteBody(r *http.Request) (postWriteBody, error) {
	var body postWriteBody
	if r == nil || r.Body == nil {
		return body, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return body, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return body, nil
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return body, fmt.Errorf("invalid JSON body")
	}
	return body, nil
}

func rawOrRendered(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		Raw      string `json:"raw"`
		Rendered string `json:"rendered"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if obj.Raw != "" {
			return obj.Raw
		}
		return obj.Rendered
	}
	return strings.Trim(string(raw), `"`)
}

type postRowInsert struct {
	Author    int64
	Date      string
	Content   string
	Title     string
	Excerpt   string
	Status    string
	Slug      string
	Type      string
	Parent    int64
	MenuOrder int // P4.157
}

// savePostRevision inserts a post_type=revision row for parentID (wp_save_post_revision subset, P4.48).
func savePostRevision(c *wp.Context, author, parentID int64, title, content, excerpt, when string) (int64, error) {
	if parentID <= 0 {
		return 0, fmt.Errorf("invalid parent")
	}
	return insertPostRow(c, postRowInsert{
		Author:  author,
		Date:    when,
		Content: content,
		Title:   title,
		Excerpt: excerpt,
		Status:  "inherit",
		Slug:    fmt.Sprintf("%d-revision-v1", parentID),
		Type:    "revision",
		Parent:  parentID,
	})
}

func insertPostRow(c *wp.Context, in postRowInsert) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	guid := fmt.Sprintf("%s/?p=0", home)
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		in.Author, in.Date, in.Date, in.Content, in.Title, in.Excerpt, in.Status,
		"open", "open", "", in.Slug, "", "", in.Date, in.Date,
		"", in.Parent, guid, in.MenuOrder, in.Type, "", 0,
	)
	if err != nil {
		return 0, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	// Fix GUID to real ID.
	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET guid = %s WHERE ID = %d",
		fmt.Sprintf("%s/?p=%d", home, id), id,
	)
	if err == nil {
		_, _ = c.DB.Exec(uq, uargs...)
	}
	return id, nil
}

func updatePostRow(c *wp.Context, id int64, title, content, excerpt, slug, status, modified string) error {
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_title = %s, post_content = %s, post_excerpt = %s, post_name = %s, post_status = %s, "+
			"post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		title, content, excerpt, slug, status, modified, modified, id, "post",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

// updatePostMenuOrder sets posts.menu_order (P4.157).
func updatePostMenuOrder(c *wp.Context, id int64, menuOrder int, modified string) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET menu_order = %d, post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		menuOrder, modified, modified, id, "post",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func updatePostStatus(c *wp.Context, id int64, status, modified string) error {
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_status = %s, post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		status, modified, modified, id, "post",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func loadPostRow(c *wp.Context, postType string, id int64) (*wp.Post, error) {
	if c == nil || c.DB == nil || id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, "+
			"post_name, post_status, post_type, post_title, post_content, post_excerpt, "+
			"guid, comment_status, ping_status, post_password, post_parent, menu_order FROM "+c.DB.Posts()+
			" WHERE ID = %d AND post_type = %s LIMIT 1",
		id, postType,
	)
	if err != nil {
		return nil, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	p := wp.Post{
		ID:            asInt64(row["ID"]),
		Author:        asInt64(row["post_author"]),
		Date:          asString(row["post_date"]),
		DateGMT:       asString(row["post_date_gmt"]),
		Modified:      asString(row["post_modified"]),
		ModifiedGMT:   asString(row["post_modified_gmt"]),
		Name:          asString(row["post_name"]),
		Status:        asString(row["post_status"]),
		Type:          asString(row["post_type"]),
		Title:         asString(row["post_title"]),
		Content:       asString(row["post_content"]),
		Excerpt:       asString(row["post_excerpt"]),
		GUID:          asString(row["guid"]),
		CommentStatus: asString(row["comment_status"]),
		PingStatus:    asString(row["ping_status"]),
		Password:      asString(row["post_password"]),
		Parent:        asInt64(row["post_parent"]),
		MenuOrder:     int(asInt64(row["menu_order"])),
	}
	return &p, nil
}

func getPostByIDAny(c *wp.Context, postType string, id int64, edit bool) (*Post, error) {
	row, err := loadPostRow(c, postType, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	p := postFromWP(c, *row, home, edit, stickyPostIDs(c))
	return &p, nil
}
