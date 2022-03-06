package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ApprovalStatus string

type ApprovalStatusValuesType struct {
	Approved ApprovalStatus
	Rejected ApprovalStatus
	Pending  ApprovalStatus
}

var ApprovalStatusValues = ApprovalStatusValuesType{
	Approved: "Approved",
	Rejected: "Rejected",
	Pending:  "Pending",
}

type ApprovalDecision string

type ApprovalDecisionValuesType struct {
	Approve ApprovalDecision
	Reject  ApprovalDecision
}

var ApprovalDecisionValues = ApprovalDecisionValuesType{
	Approve: "approve",
	Reject:  "reject",
}

type ApprovalDecisionRecord struct {
	Approver string           `json:"approver"`
	Decision ApprovalDecision `json:"decision"`
}

type ApprovalRequest struct {
	Id        int                      `json:"id,omitempty"`
	Requester string                   `json:"requester"`
	Subject   string                   `json:"subject"`
	Archived  bool                     `json:"archived"`
	Status    ApprovalStatus           `json:"status,omitempty"`
	Decisions []ApprovalDecisionRecord `json:"decisions,omitempty"`
}

var currentID int
var approvalDB map[int]*ApprovalRequest

func newID() int {
	id := currentID
	currentID += 1
	return id
}

func RequestApproval(w http.ResponseWriter, r *http.Request) {

	var ar ApprovalRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ar)
	if err != nil {
		fmt.Printf("error decoding request")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	ar.Id = newID()
	ar.Status = ApprovalStatusValues.Pending
	ar.Archived = false
	ar.Decisions = nil
	approvalDB[ar.Id] = &ar

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(ar)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func ArchiveApproval(w http.ResponseWriter, r *http.Request) {
	// get id as int from url path
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ar, ok := approvalDB[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ar.Archived = true

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(ar)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func ListApprovalRequests(w http.ResponseWriter, r *http.Request) {
	var arList []*ApprovalRequest
	for _, v := range approvalDB {
		arList = append(arList, v)
	}
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(arList)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func GetApprovalRequest(w http.ResponseWriter, r *http.Request) {
	// get id as int from url path
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ar, ok := approvalDB[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(ar)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func updateStatus(id int) {
	// id existence should be checked by the calling function
	ar, _ := approvalDB[id]
	for _, v := range ar.Decisions {
		if v.Decision == ApprovalDecisionValues.Approve {
			ar.Status = ApprovalStatusValues.Approved // a single approve is enough
		} else {
			ar.Status = ApprovalStatusValues.Rejected
			return // any reject wins over approves
		}
	}
}

func DecideOnApprovalRequest(w http.ResponseWriter, r *http.Request) {
	// get id as int from url path
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ar, ok := approvalDB[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var adr ApprovalDecisionRecord
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&adr)
	if err != nil {
		fmt.Printf("error decoding request")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	ar.Decisions = append(ar.Decisions, adr)
	updateStatus(ar.Id)
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(ar)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func main() {

	currentID = 1
	approvalDB = map[int]*ApprovalRequest{}

	r := mux.NewRouter()
	r.HandleFunc("/approval_requests", ListApprovalRequests).Methods("GET")
	r.HandleFunc("/approval_requests", RequestApproval).Methods("POST")
	r.HandleFunc("/approval_requests/{id}", ArchiveApproval).Methods("DELETE")
	r.HandleFunc("/approval_requests/{id}", GetApprovalRequest).Methods("GET")
	r.HandleFunc("/approval_requests/{id}/decisions", DecideOnApprovalRequest).Methods("POST")
	http.Handle("/", r)
	http.ListenAndServe(":5000", nil)
}
