package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/httputil"
)

type LimitAndOffsetRequest struct {
	Limit  int `form:"limit,default=100"`
	Offset int `form:"offset,default=0"`
}

type CreateProfileRequest struct {
	Key          *string `json:"key" binding:"required,notempty"`
	CurrencyCode *int    `json:"currencycode" binding:"required"`
	Description  *string `json:"description" binding:"omitempty,notempty"`
}

type UpdateProfileRequest struct {
	Key          *string `json:"key" binding:"omitempty,required_without_all=CurrencyCode Description,notempty"`
	CurrencyCode *int    `json:"currencycode" binding:"required_without_all=Key Description"`
	Description  *string `json:"description" binding:"omitempty,required_without_all=Key CurrencyCode,notempty"`
}

type profileHandler struct {
	loggerFunc    repository.LoggerFunc
	store         repository.ProfileRepository
	currencyStore repository.CurrencyRepository
}

// @BasePath /api/v1

// DeleteProfileHandler godoc
// @Summary     Delete a profile
// @Description Delete by profile ID
// @Tags        profiles
// @Accept      json
// @Produce     json
// @Param       pid    path     int true "Profile ID" Format(int64)
// @Success     200    {object} repository.Profile
// @Failure     400    {object} httputil.HTTPError
// @Failure     404    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /profiles/{pid} [delete]
func (ph *profileHandler) DeleteProfileHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	profile := &repository.Profile{Id: &id}

	err, notfound := ph.store.Delete(c, profile)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Profile with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// @BasePath /api/v1

// GetProfileHandler godoc
// @Summary     Show a profile
// @Description Get profile by ID
// @Tags        profiles
// @Accept      json
// @Produce     json
// @Param       pid    path     int true "Profile ID" Format(int64)
// @Success     200    {object} repository.Profile
// @Failure     400    {object} httputil.HTTPError
// @Failure     404    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /profiles/{pid} [get]
func (ph *profileHandler) GetProfileHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, profiles := ph.store.Query(c, repository.NewProfileSpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(profiles) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Profile with id=%d not found", id))
		return
	}

	c.JSON(http.StatusOK, profiles[0])
}

// @BasePath /api/v1

// PatchProfileHandler godoc
// @Summary     Update a profile
// @Description Update by profile ID
// @Tags        profiles
// @Accept      json
// @Produce     json
// @Param       pid     path     int                  true "Profile ID" Format(int64)
// @Param       profile body     UpdateProfileRequest true "Update profile"
// @Success     200     {object} repository.Profile
// @Failure     400     {object} httputil.HTTPError
// @Failure     404     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}  [patch]
func (ph *profileHandler) PatchProfileHandler(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	var currency *repository.Currency

	if req.CurrencyCode != nil {
		err, _, currencies := ph.currencyStore.Query(c, repository.NewCurrencySpecificationByNumericCode(*req.CurrencyCode))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(currencies) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Currency with CurrencyCode=%v not found", *req.CurrencyCode))
			return
		}
		currency = currencies[0]
	}

	profile := &repository.Profile{
		Id:          &id,
		Key:         req.Key,
		Description: req.Description,
		Currency:    currency,
	}

	err, notfound := ph.store.Update(c, profile)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Profile with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// @BasePath /api/v1

// CreateProfileHandler godoc
// @Summary     Create a profile
// @Description Add profile by json
// @Tags        profiles
// @Accept      json
// @Produce     json
// @Param       profile body     CreateProfileRequest true "Add profile"
// @Success     200     {object} repository.Profile
// @Failure     400     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles        [post]
func (ph *profileHandler) CreateProfileHandler(c *gin.Context) {
	var req CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, currencies := ph.currencyStore.Query(c, repository.NewCurrencySpecificationByNumericCode(*req.CurrencyCode))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(currencies) == 0 {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Currency with CurrencyCode=%v not found", *req.CurrencyCode))
		return
	}

	profile := &repository.Profile{
		Key:         req.Key,
		Description: req.Description,
		Currency:    currencies[0],
	}
	if err := ph.store.Add(c, profile); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// @BasePath /api/v1

// GetProfilesHandler godoc
// @Summary     List profiles
// @Description Get profiles
// @Tags        profiles
// @Accept      json
// @Produce     json
// @Param       limit  query    int false "Maximum items to return"      default(100)
// @Param       offset query    int false "Return items from this point" default(0)
// @Success     200    {object} httputil.ProfilesResponse
// @Failure     400    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /profiles       [get]
func (ph *profileHandler) GetProfilesHandler(c *gin.Context) {
	var req LimitAndOffsetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, profiles := ph.store.Query(c, repository.NewProfileSpecificationWithLimitAndOffset(
		req.Limit,
		req.Offset,
	))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	httputil.ProfilesList(c, http.StatusOK, overall, profiles)
}

func NewProfileHandler(
	store repository.ProfileRepository,
	currencyStore repository.CurrencyRepository,
	loggerFunc repository.LoggerFunc,
) *profileHandler {
	return &profileHandler{
		loggerFunc:    loggerFunc,
		store:         store,
		currencyStore: currencyStore,
	}
}
