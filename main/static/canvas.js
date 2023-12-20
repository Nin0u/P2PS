var canvas = document.getElementById("cvs");
var ctx = canvas.getContext("2d");
var intervalID;

var nb = 12;
var rotate_angle = 0;

function startWait() {
    intervalID = setInterval(draw, 1000 / 60);

}

function stopWait() {
    clearInterval(intervalID)
}

function min(a, b) {
    if (a < b) return a;
    return b;
}

function draw() {
    //ctx.fillStyle = "";
    canvas.width = window.innerWidth * 3 / 5;
    canvas.height = window.innerHeight * 3 / 5;
    ctx.clearRect(0,0,canvas.width,canvas.height);

    min_size = min(canvas.width, canvas.height);

    ctx.lineWidth = 5;

    ctx.translate(canvas.width / 2, canvas.height / 2);
    
    ctx.strokeStyle = "white";
    ctx.fillStyle = "white";

    let ampl1 = min_size / 25;
    let ampl2 = 3 * min_size / 16;

    for(i = 0; i < nb; i++) {
        let x = Math.cos(2 * Math.PI * i / nb + rotate_angle);
        let y = Math.sin(2 * Math.PI * i / nb + rotate_angle);
        ctx.beginPath();
        ctx.arc(x * ampl1, y * ampl2, 5 / 2, 0, Math.PI * 2, true);
        ctx.closePath();
        ctx.fill();
        ctx.beginPath();
        ctx.arc(x * ampl2, y * ampl1, 5 / 2, 0, Math.PI * 2, true);
        ctx.closePath();
        ctx.fill();
        ctx.rotate(Math.PI / 4);
        ctx.beginPath();
        ctx.arc(x * ampl1, y * ampl2, 5 / 2, 0, Math.PI * 2, true);
        ctx.closePath();
        ctx.fill();
        ctx.beginPath();
        ctx.arc(x * ampl2, y * ampl1, 5 / 2, 0, Math.PI * 2, true);
        ctx.closePath();
        ctx.fill();
        ctx.rotate(3 * Math.PI / 4);
    }

    rotate_angle += Math.PI * 2 / 1000;
}