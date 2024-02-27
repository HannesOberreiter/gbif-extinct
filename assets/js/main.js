
/* Set hidden sorting fields and trigger htmx */
function setSortingFields(order_by, order_dir) {
    console.info('setSortingFields', order_by, order_dir);
    document.getElementsByName('order_by')[0].value = order_by;
    document.getElementsByName('order_dir')[0].value = order_dir;
    htmx.trigger('#filterForm', 'submit');
}

/* Update form fields with url query params */
function updateFields(){
    const url = new URL(window.location.href);
    const searchParams = url.searchParams;
    console.info('updateFields', url.search);
    const filterForm = document.getElementById('filterForm');
    const inputs = filterForm.querySelectorAll('input, select');
    inputs.forEach(input => {
        if (searchParams.has(input.name)) {
            input.value = searchParams.get(input.name);
        }
    });
}

window.onload = function () {
    updateFields();
};

document.body.addEventListener("showMessage", function(evt){
   if(evt.detail.level === "error"){
     alert(evt.detail.message);   
   }
})