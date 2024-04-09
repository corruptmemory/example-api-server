// Path: /app.js

const timeDelta = 2000;

function bID(name) {
    return document.getElementById(name);
}

function clearElement(parent) {
    while (parent.firstChild) {
        parent.removeChild(parent.firstChild);
    }
}

function tableCell(row, body, cls) {
    let c = document.createElement("td");
    c.className = cls;
    c.innerHTML = body;
    row.appendChild(c);
    return c;
}

function deleteTableCell(row, id, cls) {
    let c = document.createElement("td");
    c.className = cls;
    let b = document.createElement("button");
    b.className = "delete-button";
    b.innerHTML = "Delete";
    b.onclick = function () {
        fetch(`/api/contact/${id}`, {
            method: 'DELETE'
        }).then(response => {
            if (response.status === 200) {
                renderHomePage();
            }
        });
    };
    c.appendChild(b);
    row.appendChild(c);
    return c;
}

function generateConnectionError(parent, msg) {
    clearElement(parent);
    parent.innerHTML = msg;
}




function generateContactRow(row, data) {
    tableCell(row, data.firstName, null);
    tableCell(row, data.lastName, null);
    tableCell(row, data.email, null);
    deleteTableCell(row, data.id, null);
}

function generateContacts(parent, data) {
    clearElement(parent);
    if (data == null) {
        return;
    }
    data.forEach(function (item, index) {
        let r = document.createElement("tr");
        generateContactRow(r, item);
        parent.appendChild(r);
    });
}

function renderHomePageNext() {
    setTimeout(renderHomePageLoop, timeDelta);
}

function renderServerTimeNext() {
    setTimeout(renderServerTimeLoop, 1000);
}

function renderServerTime() {
    let st = bID('server-time');
    fetch("/api/server-time")
        .catch(function (error) {
            console.log("Could not get data from server");
            generateConnectionError(dp, "Could not get data from server");
            throw error;
        }).then(response => response.json())
        .then(j => {
            st.innerText = j.time;
        });
}

function renderServerTimeLoop() {
    renderServerTime();
    renderServerTimeNext();
}


function renderHomePage() {
    let dp = bID('dashboard-parent');
    let contactsBody = bID('contacts-body');
    fetch("/api/contacts")
        .catch(function (error) {
            console.log("Could not get data from server");
            generateConnectionError(dp, "Could not get data from server");
            throw error;
        }).then(response => response.json())
        .then(j => {
            generateContacts(contactsBody, j);
        });
}



function renderHomePageLoop() {
    renderHomePage();
    renderHomePageNext();
}

function prepForm() {
    let submitButton = bID('submit-button');
    let form = bID('add-contact-form');
    let status = bID('status');
    submitButton.onclick = function () {
        let formData = new FormData(form);
        let xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/add-contact', true);
        xhr.onload = function() {
            if (xhr.status === 200) {
                // Clear the form fields
                form.reset();
                status.innerText = 'Contact added successfully';
                status.className = '';
                renderHomePage();
            } else {
                // Handle the error...
                var response = JSON.parse(xhr.responseText);
                status.className = 'error';
                status.innerText = 'Error: ' + response.error;
            }
        };
        xhr.onerror = function() {
            status.innerText = 'Error: ' + xhr.statusText;
        };
        xhr.send(formData);
    }
}
