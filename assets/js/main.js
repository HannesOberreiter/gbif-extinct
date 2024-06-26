
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
    const checkboxes = filterForm.querySelectorAll('input[type="checkbox"]');
    checkboxes.forEach(checkbox => {
        if (searchParams.has(checkbox.name)) {
            checkbox.checked = searchParams.get(checkbox.name) === "true";
        }
    });
}

/* Handle download button logic */
const downloadButton = document.getElementById('downloadBtn');
downloadButton.addEventListener('click', onDownloadClick);
function onDownloadClick() {
    if (!confirm('Do you want to the search result as csv file (max. 1_000 rows)?')) {
        return;
    }
    const downloadUrl = "/download" + new URL(window.location.href).search;
    console.info('onDownloadClick', downloadUrl);
    window.open(downloadUrl, '_blank');
}


window.onload = function () {
    updateFields();
};


document.body.addEventListener("showMessage", function(evt){
   if(evt.detail.level === "error"){
     alert(evt.detail.message);   
   }
})