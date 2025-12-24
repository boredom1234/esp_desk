




function uploadFile(file) {
  if (!file || !file.type.startsWith("image/")) {
    setUploadStatus("error", "Invalid file type");
    return;
  }

  const formData = new FormData();
  formData.append("file", file);

  
  const maxFrames =
    parseInt(document.getElementById("gifMaxFramesSlider").value) || 10;
  formData.append("maxFrames", maxFrames.toString());

  
  setUploadStatus("uploading", "Uploading...");
  document.getElementById("dropZone").classList.add("uploading");

  
  authFetch("/api/upload", {
    method: "POST",
    body: formData,
  })
    .then((res) => {
      if (!res.ok) throw new Error("Upload failed");
      return res.json();
    })
    .then((data) => {
      
      setUploadStatus("success", `${data.frameCount} frame(s)`);
      clearUploadPreview();
      loadSettings();
      if (data.frameCount > 1) {
        startAutoPlay();
      } else {
        loadCurrent();
      }

      
      if (data.bitmap) {
        lastUploadedImage = {
          bitmap: data.bitmap,
          width: data.width,
          height: data.height,
        };
        document.getElementById("saveToCycleBtn").style.display =
          "inline-block";
      } else {
        lastUploadedImage = null;
        document.getElementById("saveToCycleBtn").style.display = "none";
      }
    })
    .catch((err) => {
      if (err.message === "Unauthorized") {
        setUploadStatus("error", "Login required");
      } else {
        setUploadStatus("error", "Upload failed");
      }
    })
    .finally(() => {
      document.getElementById("dropZone").classList.remove("uploading");
    });
}


function setUploadStatus(state, text) {
  const badge = document.getElementById("uploadStatus");
  badge.className = "badge";
  badge.classList.add(state);
  badge.textContent = text;

  
  if (state !== "uploading") {
    setTimeout(() => {
      badge.className = "badge";
      badge.textContent = "Ready";
    }, 3000);
  }
}


function showUploadPreview(file) {
  const preview = document.getElementById("uploadPreview");
  const thumbnail = document.getElementById("previewThumbnail");
  const fileName = document.getElementById("previewFileName");

  fileName.textContent = file.name || "Pasted image";

  const reader = new FileReader();
  reader.onload = (e) => {
    thumbnail.src = e.target.result;
    preview.style.display = "flex";
  };
  reader.readAsDataURL(file);
}


function clearUploadPreview() {
  const fileInput = document.getElementById("imageUpload");
  fileInput.value = "";
  document.getElementById("uploadPreview").style.display = "none";
  document.getElementById("previewThumbnail").src = "";
}




function initDragAndDrop() {
  const dropZone = document.getElementById("dropZone");
  const fileInput = document.getElementById("imageUpload");

  
  dropZone.addEventListener("click", () => fileInput.click());

  
  dropZone.addEventListener("keypress", (e) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      fileInput.click();
    }
  });

  
  fileInput.addEventListener("change", () => {
    if (fileInput.files && fileInput.files[0]) {
      showUploadPreview(fileInput.files[0]);
    }
  });

  
  dropZone.addEventListener("dragenter", handleDragEnter);
  dropZone.addEventListener("dragover", handleDragOver);
  dropZone.addEventListener("dragleave", handleDragLeave);
  dropZone.addEventListener("drop", handleDrop);

  
  document.addEventListener("dragover", (e) => e.preventDefault());
  document.addEventListener("drop", (e) => e.preventDefault());
}

function handleDragEnter(e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.add("drag-over");
}

function handleDragOver(e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.add("drag-over");
}

function handleDragLeave(e) {
  e.preventDefault();
  e.stopPropagation();
  
  if (!this.contains(e.relatedTarget)) {
    this.classList.remove("drag-over");
  }
}

function handleDrop(e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.remove("drag-over");

  const files = e.dataTransfer.files;
  if (files && files.length > 0) {
    const file = files[0];
    if (file.type.startsWith("image/")) {
      
      const fileInput = document.getElementById("imageUpload");
      const dataTransfer = new DataTransfer();
      dataTransfer.items.add(file);
      fileInput.files = dataTransfer.files;
      showUploadPreview(file);
    } else {
      setUploadStatus("error", "Not an image");
    }
  }
}




function initClipboardPaste() {
  document.addEventListener("paste", handlePaste);
}

function handlePaste(e) {
  const items = e.clipboardData?.items;
  if (!items) return;

  for (const item of items) {
    if (item.type.startsWith("image/")) {
      e.preventDefault();
      const file = item.getAsFile();
      if (file) {
        
        const fileInput = document.getElementById("imageUpload");
        const dataTransfer = new DataTransfer();
        dataTransfer.items.add(file);
        fileInput.files = dataTransfer.files;
        showUploadPreview(file);

        
        document.getElementById("dropZone").focus();
      }
      break;
    }
  }
}
