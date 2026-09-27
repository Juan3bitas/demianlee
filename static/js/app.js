(function () {
    "use strict";

    // ---------- Estado global de la aplicación ----------
    const state = {
        isAdmin: !!(window.__INITIAL__ && window.__INITIAL__.isAdmin),
        hostname: (window.__INITIAL__ && window.__INITIAL__.hostname) || "",
        serverName: (window.__INITIAL__ && window.__INITIAL__.serverName) || "",
        view: "home", // home | detail | cover
        mode: "browse", // browse | select-edit | select-delete
        query: "",
        page: 1,
        totalPages: 1,
        selectedBookId: null,
    };

    const app = document.getElementById("app");

    // ---------- Helpers de API ----------
    async function apiGet(url) {
        const res = await fetch(url, { credentials: "same-origin" });
        if (!res.ok) throw await res.json().catch(() => ({ error: "Error inesperado" }));
        return res.json();
    }

    async function apiJSON(url, method, body) {
        const res = await fetch(url, {
            method,
            credentials: "same-origin",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw data;
        return data;
    }

    async function apiForm(url, method, formData) {
        const res = await fetch(url, { method, credentials: "same-origin", body: formData });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw data;
        return data;
    }

    function escapeHTML(str) {
        return String(str ?? "").replace(/[&<>"']/g, (c) => ({
            "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
        }[c]));
    }

    // ---------- Encabezado / pie de página ----------
    function renderChrome() {
        document.getElementById("welcome-text").textContent = "Bienvenido, " + state.hostname;
        document.getElementById("server-badge").textContent = "Servidor: " + state.serverName;
    }

    // ---------- Modales de Bootstrap ----------
    function openOverlay(id) {
        const el = document.getElementById(id);
        bootstrap.Modal.getOrCreateInstance(el).show();
    }

    function closeOverlay(id) {
        const el = document.getElementById(id);
        const instance = bootstrap.Modal.getInstance(el);
        if (instance) instance.hide();
    }

    function closeAllOverlays() {
        document.querySelectorAll(".modal.show").forEach((el) => {
            const instance = bootstrap.Modal.getInstance(el);
            if (instance) instance.hide();
        });
        getAdminDropdown().hide();
    }

    // ---------- Menú de administrador (Bootstrap Dropdown) ----------
    let adminDropdownInstance = null;
    function getAdminDropdown() {
        if (!adminDropdownInstance) {
            adminDropdownInstance = new bootstrap.Dropdown(document.getElementById("user-icon-btn"));
        }
        return adminDropdownInstance;
    }

    // ---------- Navegación ----------
    function goHome() {
        state.view = "home";
        state.mode = "browse";
        state.query = "";
        state.page = 1;
        state.selectedBookId = null;
        document.getElementById("search-input").value = "";
        closeAllOverlays();
        renderView();
    }

    document.getElementById("brand-link").addEventListener("click", goHome);

    function goToDetail(bookId) {
        state.view = "detail";
        state.selectedBookId = bookId;
        renderView();
    }

    function goToCover(bookId) {
        state.view = "cover";
        state.selectedBookId = bookId;
        renderView();
    }

    function startSelectMode(mode) {
        state.mode = mode; // 'select-edit' | 'select-delete'
        state.view = "home";
        state.page = 1;
        renderView();
    }

    // ---------- Búsqueda ----------
    let searchDebounce = null;
    function onSearchInput() {
        clearTimeout(searchDebounce);
        searchDebounce = setTimeout(() => {
            state.query = document.getElementById("search-input").value.trim();
            state.page = 1;
            if (state.view !== "home") state.view = "home";
            renderView();
        }, 300);
    }
    document.getElementById("search-input").addEventListener("input", onSearchInput);
    document.getElementById("search-btn").addEventListener("click", () => {
        clearTimeout(searchDebounce);
        state.query = document.getElementById("search-input").value.trim();
        state.page = 1;
        state.view = "home";
        renderView();
    });

    function arrowSVG(left) {
        const path = left ? "M10 12 6 8l4-4" : "M6 4l4 4-4 4";
        return '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="' + path + '"/></svg>';
    }

    // ---------- Vista: grilla (Home / seleccionar libro) ----------
    async function renderGridView() {
        let titleText = "Página " + state.page + " de " + state.totalPages;
        if (state.mode === "select-edit") titleText = "Seleccione el libro que quiere editar";
        if (state.mode === "select-delete") titleText = "Seleccione el libro que quiere eliminar";

        app.innerHTML =
            '<div class="d-flex align-items-center justify-content-between mb-3 dl-view-header">' +
            '<p class="dl-view-title mb-0">' + escapeHTML(titleText) + "</p>" +
            '<div class="btn-group dl-pagination-arrows" role="group">' +
            '<button class="btn btn-sm dl-arrow-btn" id="prev-page" title="Página anterior">' + arrowSVG(true) + "</button>" +
            '<button class="btn btn-sm dl-arrow-btn" id="next-page" title="Página siguiente">' + arrowSVG(false) + "</button>" +
            "</div>" +
            "</div>" +
            '<div class="row row-cols-2 row-cols-sm-3 row-cols-md-4 row-cols-xl-6 g-3" id="grid-container">' +
            '<div class="col-12 text-center py-5"><div class="spinner-border text-primary" role="status"></div></div>' +
            "</div>";

        document.getElementById("prev-page").addEventListener("click", () => {
            if (state.page > 1) { state.page -= 1; renderView(); }
        });
        document.getElementById("next-page").addEventListener("click", () => {
            if (state.page < state.totalPages) { state.page += 1; renderView(); }
        });

        try {
            const params = new URLSearchParams({ query: state.query, page: state.page });
            const data = await apiGet("/api/books?" + params.toString());
            state.totalPages = data.totalPages || 1;

            // Vuelve a pintar el número de página por si cambió tras la respuesta.
            const titleEl = app.querySelector(".dl-view-title");
            if (state.mode === "browse") {
                titleEl.textContent = "Página " + state.page + " de " + state.totalPages;
            }

            document.getElementById("prev-page").disabled = state.page <= 1;
            document.getElementById("next-page").disabled = state.page >= state.totalPages;

            const grid = document.getElementById("grid-container");
            if (!data.books || data.books.length === 0) {
                grid.innerHTML = '<div class="col-12"><div class="alert alert-secondary text-center dl-empty-state mb-0">No se encontraron libros.</div></div>';
                return;
            }

            grid.innerHTML = data.books.map((b) =>
                '<div class="col">' +
                '<button type="button" class="dl-cover-tile btn p-0 w-100" data-id="' + b.id + '" title="' + escapeHTML(b.titulo) + '">' +
                '<img src="' + escapeHTML(b.imagen) + '" class="dl-cover-img" alt="Portada de ' + escapeHTML(b.titulo) + '" loading="lazy">' +
                "</button>" +
                "</div>"
            ).join("");

            grid.querySelectorAll(".dl-cover-tile").forEach((tile) => {
                tile.addEventListener("click", () => {
                    const id = tile.getAttribute("data-id");
                    if (state.mode === "browse") {
                        goToDetail(id);
                    } else if (state.mode === "select-edit") {
                        openEditModal(id);
                    } else if (state.mode === "select-delete") {
                        openDeleteModal(id);
                    }
                });
            });
        } catch (err) {
            document.getElementById("grid-container").innerHTML =
                '<div class="col-12"><div class="alert alert-danger text-center dl-error-banner mb-0">' + escapeHTML(err.error || "No se pudo cargar el catálogo") + "</div></div>";
        }
    }

    // ---------- Vista: detalle de libro ----------
    async function renderDetailView() {
        app.innerHTML = '<div class="text-center py-5"><div class="spinner-border text-primary" role="status"></div></div>';
        try {
            const book = await apiGet("/api/books/" + encodeURIComponent(state.selectedBookId));
            app.innerHTML =
                '<div class="row g-4 align-items-start dl-detail">' +
                '<div class="col-12 col-sm-6 col-md-4 col-lg-3">' +
                '<div class="dl-detail-cover ratio ratio-2x3"><img src="' + escapeHTML(book.imagen) + '" class="object-fit-cover w-100 h-100" alt="Portada de ' + escapeHTML(book.titulo) + '"></div>' +
                "</div>" +
                '<div class="col-12 col-sm-6 col-md-8 dl-detail-info">' +
                "<h1>" + escapeHTML(book.titulo) + "</h1>" +
                "<p>" + escapeHTML(book.anio) + "</p>" +
                "<p>" + escapeHTML(book.autor) + "</p>" +
                '<button type="button" class="btn dl-btn dl-detail-btn" id="ver-libro-btn">Ver libro</button>' +
                "</div>" +
                "</div>";

            document.getElementById("ver-libro-btn").addEventListener("click", () => goToCover(book.id));
        } catch (err) {
            app.innerHTML = '<div class="alert alert-danger text-center dl-error-banner">' + escapeHTML(err.error || "Libro no encontrado") + "</div>";
        }
    }

    // ---------- Vista: texto o PDF del libro ("Ver libro") ----------
    async function renderCoverView() {
        app.innerHTML = '<div class="text-center py-5"><div class="spinner-border text-primary" role="status"></div></div>';
        try {
            const book = await apiGet("/api/books/" + encodeURIComponent(state.selectedBookId));

            if (!book.texto) {
                app.innerHTML =
                    '<div class="d-flex justify-content-center dl-cover-view">' +
                    '<div class="dl-cover-view-frame dl-text-frame shadow-sm">' +
                    '<pre class="dl-book-text">Este libro todavía no tiene un archivo de texto asociado.</pre>' +
                    "</div>" +
                    "</div>";
                return;
            }

            const isPDF = /\.pdf($|\?)/i.test(book.texto);

            if (isPDF) {
                // Los PDF se muestran embebidos con el visor nativo del navegador.
                app.innerHTML =
                    '<div class="d-flex justify-content-center dl-cover-view">' +
                    '<div class="dl-cover-view-frame dl-pdf-frame shadow-sm">' +
                    '<embed src="' + escapeHTML(book.texto) + '" type="application/pdf" class="dl-pdf-embed">' +
                    '<p class="dl-pdf-fallback">Tu navegador no puede mostrar el PDF aquí. ' +
                    '<a href="' + escapeHTML(book.texto) + '" target="_blank" rel="noopener">Ábrelo en una pestaña nueva</a>.</p>' +
                    "</div>" +
                    "</div>";
                return;
            }

            // Cualquier otro formato (por ahora .txt) se descarga y se muestra como texto plano.
            let textContent = "No se pudo cargar el texto del libro.";
            try {
                const res = await fetch(book.texto, { credentials: "same-origin" });
                if (res.ok) textContent = await res.text();
            } catch (e) { /* se mantiene el mensaje de error */ }

            app.innerHTML =
                '<div class="d-flex justify-content-center dl-cover-view">' +
                '<div class="dl-cover-view-frame dl-text-frame shadow-sm">' +
                '<pre class="dl-book-text">' + escapeHTML(textContent) + "</pre>" +
                "</div>" +
                "</div>";
        } catch (err) {
            app.innerHTML = '<div class="alert alert-danger text-center dl-error-banner">' + escapeHTML(err.error || "Libro no encontrado") + "</div>";
        }
    }

    // ---------- Enrutador de vistas ----------
    function renderView() {
        if (state.view === "home") renderGridView();
        else if (state.view === "detail") renderDetailView();
        else if (state.view === "cover") renderCoverView();
    }

    // ---------- Ícono de usuario / sesión de administrador ----------
    document.getElementById("user-icon-btn").addEventListener("click", async () => {
        try {
            const session = await apiGet("/api/session");
            state.isAdmin = session.isAdmin;
        } catch (e) { /* se asume no admin si falla */ }

        if (state.isAdmin) {
            getAdminDropdown().show();
        } else {
            document.getElementById("login-password").value = "";
            document.getElementById("login-error").textContent = "";
            openOverlay("modal-login");
        }
    });

    document.getElementById("login-submit").addEventListener("click", async () => {
        const password = document.getElementById("login-password").value;
        try {
            await apiJSON("/api/login", "POST", { password });
            state.isAdmin = true;
            closeOverlay("modal-login");
            getAdminDropdown().show();
        } catch (err) {
            document.getElementById("login-error").textContent = err.error || "No se pudo iniciar sesión";
        }
    });

    document.getElementById("login-password").addEventListener("keydown", (e) => {
        if (e.key === "Enter") document.getElementById("login-submit").click();
    });

    document.getElementById("menu-logout").addEventListener("click", async () => {
        try { await apiJSON("/api/logout", "POST", {}); } catch (e) { /* ignore */ }
        state.isAdmin = false;
        getAdminDropdown().hide();
        goHome();
    });

    document.getElementById("menu-add").addEventListener("click", () => {
        getAdminDropdown().hide();
        openCreateModal();
    });

    document.getElementById("menu-edit").addEventListener("click", () => {
        getAdminDropdown().hide();
        startSelectMode("select-edit");
    });

    document.getElementById("menu-delete").addEventListener("click", () => {
        getAdminDropdown().hide();
        startSelectMode("select-delete");
    });

    // ---------- Crear libro ----------
    function clearFieldErrors(prefix) {
        ["titulo", "autor", "anio", "imagen", "texto"].forEach((f) => {
            const el = document.getElementById(prefix + "-" + f + "-error");
            if (el) el.textContent = "";
        });
    }

    function openCreateModal() {
        document.getElementById("create-titulo").value = "";
        document.getElementById("create-autor").value = "";
        document.getElementById("create-anio").value = "";
        document.getElementById("create-imagen").value = "";
        document.getElementById("create-imagen-label").textContent = "Seleccionar portada del libro...";
        document.getElementById("create-texto").value = "";
        document.getElementById("create-texto-label").textContent = "Seleccionar texto del libro (.txt o .pdf)...";
        clearFieldErrors("create");
        openOverlay("modal-create");
    }

    document.getElementById("create-imagen").addEventListener("change", (e) => {
        const label = document.getElementById("create-imagen-label");
        label.textContent = e.target.files[0] ? e.target.files[0].name : "Seleccionar portada del libro...";
    });

    document.getElementById("create-texto").addEventListener("change", (e) => {
        const label = document.getElementById("create-texto-label");
        label.textContent = e.target.files[0] ? e.target.files[0].name : "Seleccionar texto del libro (.txt o .pdf)...";
    });

    document.getElementById("create-submit").addEventListener("click", async () => {
        clearFieldErrors("create");

        const formData = new FormData();
        formData.append("titulo", document.getElementById("create-titulo").value.trim());
        formData.append("autor", document.getElementById("create-autor").value.trim());
        formData.append("anio", document.getElementById("create-anio").value.trim());
        const file = document.getElementById("create-imagen").files[0];
        if (file) formData.append("imagen", file);
        const textFile = document.getElementById("create-texto").files[0];
        if (textFile) formData.append("texto", textFile);

        try {
            await apiForm("/api/books", "POST", formData);
            closeOverlay("modal-create");
            goHome();
        } catch (err) {
            showFieldErrors("create", err);
        }
    });

    function showFieldErrors(prefix, err) {
        if (err && err.fieldErrors) {
            Object.keys(err.fieldErrors).forEach((field) => {
                const el = document.getElementById(prefix + "-" + field + "-error");
                if (el) el.textContent = err.fieldErrors[field];
            });
        } else {
            alert((err && err.error) || "Ocurrió un error inesperado");
        }
    }

    // ---------- Editar libro ----------
    let editingBookId = null;

    function markTouched(inputEl) {
        inputEl.dataset.touched = "1";
    }

    async function openEditModal(bookId) {
        try {
            const book = await apiGet("/api/books/" + encodeURIComponent(bookId));
            editingBookId = bookId;

            const titulo = document.getElementById("edit-titulo");
            const autor = document.getElementById("edit-autor");
            const anio = document.getElementById("edit-anio");
            const imagenInput = document.getElementById("edit-imagen");

            titulo.value = "";
            titulo.placeholder = book.titulo;
            titulo.dataset.touched = "0";

            autor.value = "";
            autor.placeholder = book.autor;
            autor.dataset.touched = "0";

            anio.value = "";
            anio.placeholder = String(book.anio);
            anio.dataset.touched = "0";

            imagenInput.value = "";
            document.getElementById("edit-imagen-label").textContent = "Seleccionar portada del libro...";

            document.getElementById("edit-texto").value = "";
            document.getElementById("edit-texto-label").textContent = "Seleccionar texto del libro (.txt o .pdf)...";

            clearFieldErrors("edit");
            openOverlay("modal-edit");
        } catch (err) {
            alert((err && err.error) || "No se pudo cargar el libro");
        }
    }

    ["edit-titulo", "edit-autor", "edit-anio"].forEach((id) => {
        document.getElementById(id).addEventListener("input", (e) => markTouched(e.target));
    });

    document.getElementById("edit-imagen").addEventListener("change", (e) => {
        const label = document.getElementById("edit-imagen-label");
        label.textContent = e.target.files[0] ? e.target.files[0].name : "Seleccionar portada del libro...";
    });

    document.getElementById("edit-texto").addEventListener("change", (e) => {
        const label = document.getElementById("edit-texto-label");
        label.textContent = e.target.files[0] ? e.target.files[0].name : "Seleccionar texto del libro (.txt o .pdf)...";
    });

    document.getElementById("edit-submit").addEventListener("click", async () => {
        clearFieldErrors("edit");

        const titulo = document.getElementById("edit-titulo");
        const autor = document.getElementById("edit-autor");
        const anio = document.getElementById("edit-anio");
        const file = document.getElementById("edit-imagen").files[0];
        const textFile = document.getElementById("edit-texto").files[0];

        const formData = new FormData();
        if (titulo.dataset.touched === "1") {
            formData.append("touched_titulo", "1");
            formData.append("titulo", titulo.value.trim());
        }
        if (autor.dataset.touched === "1") {
            formData.append("touched_autor", "1");
            formData.append("autor", autor.value.trim());
        }
        if (anio.dataset.touched === "1") {
            formData.append("touched_anio", "1");
            formData.append("anio", anio.value.trim());
        }
        if (file) formData.append("imagen", file);
        if (textFile) formData.append("texto", textFile);

        try {
            await apiForm("/api/books/" + encodeURIComponent(editingBookId), "PUT", formData);
            closeOverlay("modal-edit");
            goHome();
        } catch (err) {
            showFieldErrors("edit", err);
        }
    });

    // ---------- Eliminar libro ----------
    let deletingBookId = null;

    async function openDeleteModal(bookId) {
        try {
            const book = await apiGet("/api/books/" + encodeURIComponent(bookId));
            deletingBookId = bookId;
            document.getElementById("delete-confirm-text").textContent =
                "¿Seguro quiere eliminar el libro " + book.titulo + "?";
            openOverlay("modal-delete-confirm");
        } catch (err) {
            alert((err && err.error) || "No se pudo cargar el libro");
        }
    }

    document.getElementById("delete-confirm-submit").addEventListener("click", async () => {
        try {
            await fetch("/api/books/" + encodeURIComponent(deletingBookId), {
                method: "DELETE",
                credentials: "same-origin",
            }).then(async (res) => {
                if (!res.ok) throw await res.json().catch(() => ({}));
            });
            closeOverlay("modal-delete-confirm");
            goHome();
        } catch (err) {
            alert((err && err.error) || "No se pudo eliminar el libro");
            closeOverlay("modal-delete-confirm");
        }
    });

    // Cierra el menú de administrador si el usuario hace clic en cualquier
    // otra parte del documento (comportamiento estándar de un dropdown).
    document.addEventListener("click", (e) => {
        const wrapper = document.querySelector(".dl-user-dropdown");
        if (adminDropdownInstance && wrapper && !wrapper.contains(e.target)) {
            adminDropdownInstance.hide();
        }
    });

    // ---------- Arranque ----------
    renderChrome();
    renderView();
})();
