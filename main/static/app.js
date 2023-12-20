var TitleBlock = document.getElementById("TitleBlock");
var DListBlock = document.getElementById("DList");
var TreeBlock  = document.getElementById("Tree");
var WaitBlock  = document.getElementById("Wait");

var ExportButton = document.getElementById("Export");
var DownloadButton = document.getElementById("Download");

var BackDL = document.getElementById("BackDL");
var UpdateDL = document.getElementById("UpdateDL");
var DownloadDL = document.getElementById("DownloadDL");

var BackTree = document.getElementById("BackTR");
var DownloadTree = document.getElementById("DownloadTR");

var ListDL = document.getElementById("ListDL");
var peername = "";
var lastPersoneChose = null;

var ContentTree = document.getElementById("Content");
var content = "";
var lastContentChose = null;

//TODO: handle error (si le peername n existe pas / si le file n existe pas / si le serveur ne réponds pas)

ExportButton.addEventListener("click", clickExport);
DownloadButton.addEventListener("click", clickDownload)

BackDL.addEventListener("click", clickBackDL)
UpdateDL.addEventListener("click", clickUpdateDL);
DownloadDL.addEventListener("click", clickDownloadDL);

BackTree.addEventListener("click", clickBackTR);
DownloadTree.addEventListener("click", clickDownloadTR);


function clickExport() {
    startWait();
    change(TitleBlock, WaitBlock);

    fetch("http://localhost:8080/export", {
        method: "POST",
        headers: {
            "Content-type": "application/json; charset=UTF-8"
        }
    }).then(response => {
        stopWait();
        change(WaitBlock, TitleBlock);
    });

}

function removeAllChildNodes(parent) {
    while (parent.firstChild) {
        parent.removeChild(parent.firstChild);
    }
}

function change(bloc1, bloc2) {
    bloc1.style.opacity = 0;
    setTimeout(function() {
        bloc1.classList.add("disp");
        bloc2.classList.remove("disp");
        setTimeout(function() {
            bloc2.style.opacity = 1;
        }, 300)
    }, 1000);
}

function dlList() {
    fetch("http://localhost:8080/peer", {
        method: "POST",
        headers: {
            "Content-type": "application/json; charset=UTF-8"
        }
    }).then(response => {
        response.json().then(data => {
            console.log(data.list);
            removeAllChildNodes(ListDL);
            for(elt of data.list) {
                let elt_html = ListDL.appendChild(document.createElement("div"));
                elt_html.textContent = elt;
                elt_html.classList.add("bouton", "active");
            }
            for(i = 0; i < ListDL.children.length; i++) {
                ListDL.children[i].addEventListener("click", (e) => {
                    console.log(e.target.textContent);

                    if (lastPersoneChose != null)
                        lastPersoneChose.classList.remove("selected");

                    e.target.classList.add("selected");
                    peername = e.target.textContent;
                    lastPersoneChose = e.target;

                    DownloadDL.classList.add("active");
                })
            }
            stopWait();
            change(WaitBlock, DListBlock);
        })
    })
}

function clickDownload() {
    if(!DownloadButton.classList.contains("active")) return;
    startWait();
    change(TitleBlock, WaitBlock);
    dlList();
}

function clickBackDL() {
    DownloadDL.classList.remove("active");
    peername = "";
    removeAllChildNodes(ListDL);
    change(DListBlock, TitleBlock);
}

function clickUpdateDL() {
    DownloadDL.classList.remove("active");
    peername = "";
    startWait();
    change(DListBlock, WaitBlock);
    removeAllChildNodes(ListDL);

    dlList();
} 


function buildTree(data, elt_html, path) {
    if(data.FileType != 2) {
        let elt = elt_html.appendChild(document.createElement("li"));
        elt.classList.add("node");
        elt.textContent = data.Name;
        elt.setAttribute("path", path + data.Name);
        elt.addEventListener("click", clickFile);
        return;
    }

    let li = elt_html.appendChild(document.createElement("li"));
    let elt = li.appendChild(document.createElement("details"));
    let summary = elt.appendChild(document.createElement("summary"));
    summary.classList.add("node");
    summary.textContent = data.Name;
    summary.setAttribute("path", path + data.Name);
    summary.addEventListener("click", clickFile);

    let ul = elt.appendChild(document.createElement("ul"));

    for(child of data.Children) {
        buildTree(child, ul, path + data.Name + "/");
    }
} 

function clickDownloadDL() {
    if(!DownloadDL.classList.contains("active")) return

    startWait();
    change(DListBlock, WaitBlock);

    fetch("http://localhost:8080/data", {
        method: "POST",
        headers: {
            "Content-type": "application/json; charset=UTF-8"
        },
        body: JSON.stringify({peer: peername}),
    }).then(response => {
        response.json().then(data => {

            removeAllChildNodes(ContentTree);
            content = "";
            lastContentChose = null;

            let elt_html = ContentTree.appendChild(document.createElement("ul"));
            elt_html.classList.add("tree");

            buildTree(data, elt_html, "");


            stopWait();
            change(WaitBlock, TreeBlock);
        });
    });

}

function clickFile(e) {
    console.log(e.target.textContent);
    if (lastContentChose != null)
        lastContentChose.classList.remove("selected");

    e.target.classList.add("selected");
    content = e.target.getAttribute("path");
    lastContentChose = e.target;

    DownloadTree.classList.add("active");
}

function clickBackTR() {
    DownloadDL.classList.remove("active");
    DownloadTree.classList.remove("active");
    peername = "";
    startWait();
    change(TreeBlock, WaitBlock);
    removeAllChildNodes(ListDL);

    dlList();
}

function clickDownloadTR() {
    startWait();
    change(TreeBlock, WaitBlock);
    fetch("http://localhost:8080/download", {
        method: "POST",
        headers: {
            "Content-type": "application/json; charset=UTF-8"
        },
        body: JSON.stringify({path: content, peer:peername})
    }).then(response => {
        console.log(response)
        console.log("END DOWNLOAD");
        stopWait();
        change(WaitBlock, TreeBlock);
    })
}