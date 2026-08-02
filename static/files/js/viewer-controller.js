/* globals app */

app.controller("ViewerController", function($scope, $rootScope, $http, $sce, $interval, $timeout) {
  $rootScope.viewer = $scope;
  $scope.mediaList = [];
  $scope.movies = [];
  $scope.tvShows = [];
  $scope.featuredMedia = null;
  $scope.selectedMedia = null;
  $scope.activePlayerFile = null;
  $scope.streamUrl = "";
  $scope.subtitlesUrl = "";
  $scope.hasSubtitles = false;

  $scope.progressInterval = null;
  $scope.lastSavedTime = 0;

  // Initialize and load items
  $scope.init = function() {
    $scope.loadMedia();
  };

  $scope.loadMedia = function() {
    $http.get("/api/media/list")
      .then(function(res) {
        $scope.mediaList = res.data || [];
        $scope.movies = $scope.mediaList.filter(function(item) { return item.media_type === 'movie'; });
        $scope.tvShows = $scope.mediaList.filter(function(item) { return item.media_type === 'tv'; });

        // Select a featured hero banner item
        if ($scope.mediaList.length > 0) {
          $scope.featuredMedia = $scope.mediaList[Math.floor(Math.random() * $scope.mediaList.length)];
        }
      }, function(err) {
        console.error("Failed to fetch media list:", err);
      });
  };

  $scope.openDetails = function(item) {
    $scope.selectedMedia = item;
    // Auto-organize files by season if TV show
    if (item.media_type === 'tv' && item.files) {
      $scope.seasons = {};
      item.files.forEach(function(file) {
        var sNum = file.season_number || 1;
        if (!$scope.seasons[sNum]) {
          $scope.seasons[sNum] = [];
        }
        $scope.seasons[sNum].push(file);
      });
    }
  };

  $scope.closeDetails = function() {
    $scope.selectedMedia = null;
  };

  $scope.startPlayback = function(file) {
    $scope.activePlayerFile = file;
    $scope.closeDetails();

    // Point to our on-the-fly transcoding and direct-streaming endpoint
    var rawUrl = "/api/stream?path=" + encodeURIComponent(file.file_path);
    $scope.streamUrl = $sce.trustAsResourceUrl(rawUrl);

    // Look for subtitles by locating potential .srt / .vtt companions in the downloads
    $scope.hasSubtitles = false;
    $scope.subtitlesUrl = "";

    // Check if subtitle exists alongside the file (assuming common .srt suffix)
    var srtPath = file.file_path.replace(/\.[a-zA-Z0-9]+$/, ".srt");
    var vttPath = file.file_path.replace(/\.[a-zA-Z0-9]+$/, ".vtt");

    // Default to pointing directly to subtitles endpoint with converted srt or native vtt
    var rawSubUrl = "/api/subtitles?path=" + encodeURIComponent(srtPath);
    $scope.subtitlesUrl = $sce.trustAsResourceUrl(rawSubUrl);
    $scope.hasSubtitles = true;

    // Fetch progress and prompt resume
    $http.get("/api/progress?path=" + encodeURIComponent(file.file_path))
      .then(function(res) {
        if (res.data && res.data.progress_seconds > 5) {
          var resumeTime = res.data.progress_seconds;
          var video = document.getElementById("netflixPlayer");
          if (video) {
            if (confirm("Would you like to resume from where you left off (" + formatTime(resumeTime) + ")?")) {
              video.currentTime = resumeTime;
            }
          }
        }
      });

    // Setup interval to track progress and sync with DB every 5 seconds
    if ($scope.progressInterval) {
      $interval.cancel($scope.progressInterval);
    }
    $scope.progressInterval = $interval(function() {
      var video = document.getElementById("netflixPlayer");
      if (video && !video.paused) {
        var currentTime = Math.floor(video.currentTime);
        if (currentTime !== $scope.lastSavedTime) {
          $scope.lastSavedTime = currentTime;
          var isWatched = video.duration ? (currentTime / video.duration > 0.9) : false;

          $http.post("/api/progress", {
            file_path: file.file_path,
            progress_seconds: currentTime,
            is_watched: isWatched
          });
        }
      }
    }, 5000);
  };

  $scope.closePlayer = function() {
    $scope.activePlayerFile = null;
    $scope.streamUrl = "";
    $scope.subtitlesUrl = "";
    if ($scope.progressInterval) {
      $interval.cancel($scope.progressInterval);
    }
  };

  function formatTime(seconds) {
    var min = Math.floor(seconds / 60);
    var sec = Math.floor(seconds % 60);
    return min + ":" + (sec < 10 ? "0" : "") + sec;
  }

  // Fallback downloads-count behavior
  $scope.numDownloads = function() {
    if ($rootScope.state && $rootScope.state.Downloads && $rootScope.state.Downloads.Children) {
      return $rootScope.state.Downloads.Children.length;
    }
    return 0;
  };

  $scope.init();
});

app.controller("ViewerNodeController", function($scope, $rootScope) {
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

  $scope.audioPreview = /\.(mp3|m4a)$/i.test(path);
  $scope.imagePreview = /\.(jpe?g|png|gif)$/i.test(path);
  $scope.videoPreview = /\.(mp4|mkv|mov|avi|webm)$/i.test(path);

  $scope.closed = function() {
    return n.$closed;
  };
  $scope.toggle = function() {
    n.$closed = !n.$closed;
  };
});
