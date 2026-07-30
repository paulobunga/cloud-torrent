/* globals app */

app.controller("ViewerController", function($scope, $rootScope, $sce) {
  $rootScope.viewer = $scope;
  $scope.currentFile = null;
  $scope.streamUrl = "";

  $scope.numDownloads = function() {
    if ($rootScope.state.Downloads && $rootScope.state.Downloads.Children) {
      return $rootScope.state.Downloads.Children.length;
    }
    return 0;
  };

  $scope.playFile = function(node) {
    $scope.currentFile = node;
    var path = node.$path;
    $scope.audioPreview = /\.(mp3|m4a)$/i.test(path);
    $scope.imagePreview = /\.(jpe?g|png|gif)$/i.test(path);
    $scope.videoPreview = /\.(mp4|mkv|mov|avi|webm)$/i.test(path);

    // Use trustAsResourceUrl to avoid Angular SCE context errors if necessary
    var rawUrl = "/download/" + encodeURIComponent(path).replace(/%2F/g, "/");
    $scope.streamUrl = $sce.trustAsResourceUrl(rawUrl);
  };
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

  // Closed by default if modified more than 24h ago
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
