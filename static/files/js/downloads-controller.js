/* globals app */

app.controller("DownloadsController", function($scope, $rootScope) {
  $rootScope.downloads = $scope;

  $scope.numDownloads = function() {
    if ($scope.state.Downloads && $scope.state.Downloads.Children)
      return $scope.state.Downloads.Children.length;
    return 0;
  };
});

app.controller("NodeController", function($scope, $rootScope, $http, $timeout) {
  var n = $scope.node;
  $scope.isfile = function() {
    return !n.Children;
  };
  $scope.isdir = function() {
    return !$scope.isfile();
  };

  var pathArray = [n.Name];
  if ($scope.$parent && $scope.$parent.$parent && $scope.$parent.$parent.node) {
    var parentNode = $scope.$parent.$parent.node;
    pathArray.unshift(parentNode.$path);
    n.$depth = parentNode.$depth + 1;
  } else {
    n.$depth = 1;
  }
  var path = (n.$path = pathArray.join("/"));
  n.$closed = $scope.agoHrs(n.Modified) > 24;
  $scope.audioPreview = /\.(mp3|m4a)$/.test(path);
  $scope.imagePreview = /\.(jpe?g|png|gif)$/.test(path);
  $scope.videoPreview = /\.(mp4|mkv|mov)$/.test(path);

  //search for this file
  var torrents = $rootScope.state.Torrents;
  if ($scope.isfile() && torrents) {
    for (var ih in torrents) {
      var torrent = torrents[ih];
      var files = torrent.Files;
      if (files) {
        for (var i = 0; i < files.length; i++) {
          var f = files[i];
          if (f.Path === path) {
            n.$torrent = torrent;
            n.$file = f;
            break;
          }
        }
      }
      if (n.$file) break;
    }
  }

  $scope.isdownloading = function() {
    return (
      n.$torrent &&
      n.$torrent.Loaded &&
      n.$torrent.Started &&
      n.$file &&
      n.$file.Percent < 100
    );
  };

  $scope.preremove = function() {
    $scope.confirm = true;
    $timeout(function() {
      $scope.confirm = false;
    }, 3000);
  };

  //defaults
  $scope.closed = function() {
    return n.$closed;
  };
  $scope.toggle = function() {
    n.$closed = !n.$closed;
  };
  $scope.icon = function() {
    var c = [];
    if ($scope.isdownloading()) {
      c.push("spinner", "loading");
    } else {
      c.push("outline");
      if ($scope.isfile()) {
        if ($scope.audioPreview) c.push("audio");
        else if ($scope.imagePreview) c.push("image");
        else if ($scope.videoPreview || /\.(avi)$/.test(path)) c.push("video");
        c.push("file");
      } else {
        c.push("folder");
        if (!$scope.closed()) c.push("open");
      }
    }
    c.push("icon");
    return c.join(" ");
  };

  $scope.remove = function() {
    $http.delete("/download/" + n.$path);
  };

  $scope.togglePreview = function() {
    $scope.showPreview = !$scope.showPreview;
  };

  $scope.openEnrichModal = function(node) {
    if ($rootScope.openEnrichModal) {
      $rootScope.openEnrichModal(node);
    }
  };
});

app.controller("EnrichController", function($scope, $rootScope, $http) {
  $scope.searchQuery = "";
  $scope.searchResults = [];
  $scope.selectedResult = null;
  $scope.enrichingNode = null;
  $scope.loading = false;
  $scope.mediaType = "movie";
  $scope.seasonNumber = "";
  $scope.episodeNumber = "";

  $rootScope.openEnrichModal = function(node) {
    $scope.enrichingNode = node;
    $scope.searchQuery = cleanTitle(node.Name);
    $scope.searchResults = [];
    $scope.selectedResult = null;
    $scope.mediaType = "movie";
    $scope.seasonNumber = "";
    $scope.episodeNumber = "";
    $scope.searchTMDB();
  };

  $scope.closeModal = function() {
    $scope.enrichingNode = null;
  };

  $scope.searchTMDB = function() {
    if (!$scope.searchQuery) return;
    $scope.loading = true;
    $scope.searchResults = [];
    $http.get("/api/tmdb/search?query=" + encodeURIComponent($scope.searchQuery))
      .then(function(res) {
        $scope.loading = false;
        if (res.data && res.data.results) {
          $scope.searchResults = res.data.results.filter(function(item) {
            return item.media_type === "movie" || item.media_type === "tv";
          });
        }
      }, function(err) {
        $scope.loading = false;
        console.error("TMDB search error:", err);
      });
  };

  $scope.selectResult = function(item) {
    $scope.selectedResult = item;
    $scope.mediaType = item.media_type || "movie";
  };

  $scope.saveEnrichment = function() {
    if (!$scope.selectedResult || !$scope.enrichingNode) return;
    $scope.loading = true;

    var payload = {
      file_path: $scope.enrichingNode.$path,
      tmdb_id: $scope.selectedResult.id,
      title: $scope.selectedResult.title || $scope.selectedResult.name,
      media_type: $scope.mediaType,
      overview: $scope.selectedResult.overview,
      poster_path: $scope.selectedResult.poster_path ? "https://image.tmdb.org/t/p/w500" + $scope.selectedResult.poster_path : "",
      backdrop_path: $scope.selectedResult.backdrop_path ? "https://image.tmdb.org/t/p/original" + $scope.selectedResult.backdrop_path : "",
      release_date: $scope.selectedResult.release_date || $scope.selectedResult.first_air_date || "",
      genres: "",
      rating: $scope.selectedResult.vote_average || 0.0
    };

    if ($scope.mediaType === "tv") {
      if ($scope.seasonNumber !== "") {
        payload.season_number = parseInt($scope.seasonNumber, 10);
      }
      if ($scope.episodeNumber !== "") {
        payload.episode_number = parseInt($scope.episodeNumber, 10);
      }
    }

    $http.post("/api/media/enrich", payload)
      .then(function() {
        $scope.loading = false;
        alert("Enrichment saved successfully!");
        $scope.closeModal();
      }, function(err) {
        $scope.loading = false;
        alert("Enrichment save error: " + (err.data || err.statusText));
      });
  };

  function cleanTitle(filename) {
    var name = filename.replace(/\.[a-zA-Z0-9]+$/, ""); // remove extension
    name = name.replace(/[_.]/g, " "); // replace dots/underscores with spaces
    name = name.replace(/\b(1080p|720p|2160p|4k|bluray|hdr|h264|x264|h265|x265|yify|brrip|webrip|dvdrip|xvid|ac3|hdrip|dual|audio|multi|web-dl|aac|dts)\b/gi, ""); // strip words
    name = name.replace(/\b(19\d{2}|20\d{2})\b/gi, ""); // strip years
    name = name.replace(/\s+/g, " ").trim(); // clean spaces
    return name;
  }
});
